/* cleand.c - JunkClean daemon: HTTP :8801 + timer + progress + AI orchestration
 * 静态编译：aarch64-linux-gnu-gcc -O2 -static -o bin/cleand cleand.c -pthread
 */
#include <stdio.h>
#include <dirent.h>
#include <stdlib.h>
#include <string.h>
#include <strings.h>
#include <unistd.h>
#include <errno.h>
#include <signal.h>
#include <stdarg.h>
#include <fcntl.h>
#include <pthread.h>
#include <time.h>
#include <sys/socket.h>
#include <sys/types.h>
#include <sys/wait.h>
#include <sys/stat.h>
#include <netinet/in.h>
#include <arpa/inet.h>

#define PORT 46780
#define MAXBUF 65536
#define MAXRES 262144

static char ADR[512]  = "/data/adb/junk-cleaner";   /* runtime data */
static char MODDIR[512] = "/data/adb/modules/junkclean"; /* module dir */
static char SH[64] = "/system/bin/sh";

/* ---- task state (progress) ---- */
static struct {
    volatile int running;
    volatile int pct;
    char msg[128];
    char result[MAXRES];
    time_t last_ts;
} task = {0, 0, "", "", 0};
static pthread_mutex_t task_mu = PTHREAD_MUTEX_INITIALIZER;

static void tsave(const char *pct, const char *msg){
    pthread_mutex_lock(&task_mu);
    task.pct = atoi(pct); if (task.pct<0) task.pct=0; if (task.pct>100) task.pct=100;
    snprintf(task.msg, sizeof(task.msg), "%s", msg);
    pthread_mutex_unlock(&task_mu);
}

/* ---- tiny HTTP helpers ---- */
static void http_head(int fd, int code, const char *ct, long len){
    char b[512];
    snprintf(b, sizeof(b), "HTTP/1.1 %d OK\r\nContent-Type: %s\r\nContent-Length: %ld\r\nAccess-Control-Allow-Origin: *\r\nCache-Control: no-store\r\nConnection: close\r\n\r\n", code, ct, len);
    write(fd, b, strlen(b));
}
static void http_json(int fd, const char *s){
    http_head(fd, 200, "application/json", (long)strlen(s));
    write(fd, s, strlen(s));
}

static void log_msg(const char *level, const char *fmt, ...);
static void cleanup_exit(int sig);
static volatile int ai_last=0;
static void cleanup_exit(int sig){ log_msg("INFO","daemon exiting (sig=%d)", sig); kill(0,SIGTERM); usleep(500000); _exit(0); }

/* ---- daemon log: $ADR/daemon.log (256KB rotate) ---- */
static void log_msg(const char *level, const char *fmt, ...){
    char path[700]; snprintf(path,sizeof(path),"%s/daemon.log",ADR);
    FILE *f = fopen(path,"a");
    /* ADR 不可写时降级到 /data/local/tmp，确保失败也有迹可查 */
    if(!f){ snprintf(path,sizeof(path),"/data/local/tmp/jc_daemon.log"); f=fopen(path,"a"); }
    if(!f) return;
    time_t now = time(NULL); struct tm *tmv = localtime(&now);
    fprintf(f,"%04d-%02d-%02d %02d:%02d:%02d [%s] ",
        tmv->tm_year+1900,tmv->tm_mon+1,tmv->tm_mday,tmv->tm_hour,tmv->tm_min,tmv->tm_sec, level);
    va_list ap; va_start(ap,fmt); vfprintf(f,fmt,ap); va_end(ap);
    fprintf(f,"\n");
    fclose(f);
    if(access(path,F_OK)==0){
        long sz = 0; FILE *sf=fopen(path,"r"); if(sf){ fseek(sf,0,SEEK_END); sz=ftell(sf); fclose(sf); }
        if(sz > 262144){
            char tmp[700]; snprintf(tmp,sizeof(tmp),"%s.old",path);
            rename(path,tmp);
            FILE *t=fopen(tmp,"r"); FILE *n=fopen(path,"w");
            if(t&&n){ fseek(t,-131072,SEEK_END); char b[8192]; size_t r;
                while((r=fread(b,1,sizeof(b),t))>0) fwrite(b,1,r,n); }
            if(t)fclose(t); if(n)fclose(n); unlink(tmp);
        }
    }
}

static void http_err(int fd, int code, const char *s){
    http_head(fd, code, "application/json", (long)strlen(s));
    write(fd, s, strlen(s));
}
/* url-decode into dst (in-place safe) */
static void urldec(char *s){
    char *d = s;
    for (; *s; s++){
        if (*s=='%' && s[1] && s[2]){
            int h1 = s[1]>='0'&&s[1]<='9'?s[1]-'0':(s[1]|32)-'a'+10;
            int h2 = s[2]>='0'&&s[2]<='9'?s[2]-'0':(s[2]|32)-'a'+10;
            *d++ = (char)(h1*16+h2); s+=2;
        } else *d++ = *s;
    }
    *d = 0;
}
/* extract key=value from JSON-ish body: {"a":"b","c":1} -> val for key (first occurrence) */
static int jget(const char *body, const char *key, char *out, int outsz){
    const char *k = strstr(body, key);
    if (!k) return -1;
    k += strlen(key);
    while (*k==' '||*k==':') k++;
    if (*k=='"'){
        k++; const char *e = strchr(k,'"'); if(!e) return -1;
        int n = (int)(e-k); if (n>=outsz) n=outsz-1;
        memcpy(out,k,n); out[n]=0; return 0;
    }
    return -1; /* numeric keys not needed here */
}

/* ---- run cleaner.sh subcommand in background thread; stream PROG; collect last JSON line ---- */
static void *runner(void *arg){
    char *cmd = (char*)arg;      /* full command line e.g. "clean cache,apk force" */
    int pipefd[2];
    if (pipe(pipefd)) { pthread_mutex_lock(&task_mu); task.running=0; pthread_mutex_unlock(&task_mu); free(arg); return NULL; }
    setenv("JC_ADR", ADR, 1);      /* cleaner.sh 继承运行时目录 */
    setenv("JC_MOD", MODDIR, 1);   /* 模块目录 */
    pid_t pid = fork();
    if (pid<0){ pthread_mutex_lock(&task_mu); task.running=0; pthread_mutex_unlock(&task_mu); close(pipefd[0]); close(pipefd[1]); free(arg); return NULL; }
    if (pid==0){
        dup2(pipefd[1],1); dup2(pipefd[1],2); close(pipefd[0]);
        char p[600]; snprintf(p,sizeof(p),"%s/cleaner.sh",MODDIR);
        execl(SH, SH, p, cmd, (char*)NULL);
        _exit(127);
    }
    close(pipefd[1]);
    log_msg("INFO","task start: %s", cmd);
    pthread_mutex_lock(&task_mu); task.running=1; task.pct=0; strcpy(task.msg,"启动…"); task.result[0]=0; pthread_mutex_unlock(&task_mu);
    /* read stream */
    char buf[1024]; ssize_t n;
    while ((n = read(pipefd[0], buf, sizeof(buf)-1)) > 0){
        buf[n]=0;
        char *nl;
        char *rest = buf;
        while ((nl = strchr(rest,'\n'))){
            *nl=0;
            if (!strncmp(rest,"PROG ",5)){ char *s=rest+5,*sp=strchr(s,' '); if(sp){*sp=0; tsave(s,sp+1);} }
            else if (rest[0] == '{') { strncpy(task.result, rest, sizeof(task.result)-1); task.result[sizeof(task.result)-1]=0; }
            rest = nl+1;
        }
    }
    close(pipefd[0]);
    int st=0; for(int w=0;w<180;w++){ if(waitpid(pid,&st,WNOHANG)==pid) break; usleep(500000); } kill(pid,SIGKILL); waitpid(pid,&st,0);
    if(WIFEXITED(st)&&WEXITSTATUS(st)!=0) log_msg("WARN","task %s exited rc=%d", cmd, WEXITSTATUS(st));
    else if(WIFSIGNALED(st)) log_msg("WARN","task %s killed by sig %d", cmd, WTERMSIG(st));
    pthread_mutex_lock(&task_mu); task.running=0; task.pct=100; snprintf(task.msg,sizeof(task.msg),"完成"); task.last_ts=time(NULL); pthread_mutex_unlock(&task_mu);
    log_msg("INFO","task done: %s", cmd);
    free(arg);
    return NULL;
}
/* spawn background task; returns 0 on start */
static int bg(const char *cmdline){
    pthread_t t;
    char *c = strdup(cmdline);
    return pthread_create(&t, NULL, runner, c);
}

/* ---- static file from webroot (no traversal) ---- */
static void serve_file(int fd, char *path){
    char full[700];
    if (strstr(path,"..") || strchr(path,'\\')){ http_err(fd,403,"{\"e\":\"forbidden\"}"); return; }
    if (!strcmp(path,"/")) { strcpy(path,"/index.html"); }
    snprintf(full,sizeof(full),"%s/webroot%s",MODDIR,path);
    FILE *f = fopen(full,"rb");
    if (!f){ http_err(fd,404,"{\"e\":\"not found\"}"); return; }
    fseek(f,0,SEEK_END); long sz = ftell(f); fseek(f,0,SEEK_SET);
    const char *ct = "text/plain";
    if (strstr(path,".html")) ct="text/html; charset=utf-8";
    else if (strstr(path,".js")) ct="application/javascript";
    else if (strstr(path,".css")) ct="text/css";
    else if (strstr(path,".png")) ct="image/png";
    else if (strstr(path,".svg")) ct="image/svg+xml";
    http_head(fd,200,ct,sz);
    char *mem = malloc(sz); 
    if (fread(mem,1,sz,f)==(size_t)sz) write(fd,mem,sz);
    free(mem); fclose(f);
}

/* ponytail: config 用 K=V 文本 + 手动解析（无类型校验）。天花板：配置错误无即时反馈（坏值静默用默认）。升级路径：解析时校验类型/范围。 */
/* ---- config file read/write ---- */
static void api_config(int fd, const char *method, const char *body){
    char p[600]; snprintf(p,sizeof(p),"%s/config.conf",ADR);
    if(getenv("JC_DEBUG")){ FILE*dbg=fopen("/tmp/jc/run/post.dbg","w"); fprintf(dbg,"method=[%.8s] body=[%.200s]\n",method,body?body:"<null>"); fclose(dbg); }
    if (!strcmp(method,"POST")){
        if (body && *body){
            /* atomic-ish write */
            char tmp[600]; snprintf(tmp,sizeof(tmp),"%s/config.conf.tmp",ADR);
            FILE *f=fopen(tmp,"w");
            if (!f){ http_err(fd,500,"{\"e\":\"write fail\"}"); return; }
            fwrite(body,1,strlen(body),f); fclose(f);
            if (rename(tmp,p)!=0){ http_err(fd,500,"{\"e\":\"rename fail\"}"); return; }
            chmod(p,0600);
            http_json(fd,"{\"ok\":1}");
            /* reload interval effects happen on next loop */
        } else http_err(fd,400,"{\"e\":\"empty body\"}");
        return;
    }
    char buf[8192]; int n=0; FILE *f=fopen(p,"r");
    if (!f){ http_json(fd,"{}"); return; }
    n = fread(buf,1,sizeof(buf)-1,f); fclose(f); buf[n]=0;
    char *out = malloc(n+64); snprintf(out,n+64,"{\"cfg\":%s}",buf);
    http_json(fd,out); free(out);
}

/* ---- rules read/write: /api/rules?type=cache|junk|... ---- */
static void api_rules(int fd, const char *method, const char *q, const char *body){
    char type[64]="cache";
    const char *tq = strstr(q,"type="); 
    if (tq){ tq+=5; const char *e=strchr(tq,'&'); int n=e?(int)(e-tq):(int)strlen(tq); if(n>63)n=63; memcpy(type,tq,n); type[n]=0; }
    char p[700];
    if (strstr(type,"..")||strchr(type,'/')||strchr(type,'\\')){ http_err(fd,400,"{\"e\":\"bad type\"}"); return; }
    snprintf(p,sizeof(p),"%s/rules/%s.list",ADR,type);
    if (!strcmp(method,"POST")){
        if (!body || !*body){ http_err(fd,400,"{\"e\":\"empty\"}"); return; }
        char bak[700]; snprintf(bak,sizeof(bak),"%s.bak",p);
        rename(p,bak);
        FILE *f=fopen(p,"w");
        if (f){ fwrite(body,1,strlen(body),f); fclose(f); http_json(fd,"{\"ok\":1}"); }
        else { rename(bak,p); http_err(fd,500,"{\"e\":\"write fail\"}"); }
        return;
    }
    char out[MAXRES]; int n=0;
    n = snprintf(out,sizeof(out),"{\"type\":\"%s\",\"content\":",type);
    FILE *f=fopen(p,"r"); if(!f){ strcat(out,"\"\""); } else {
        char *txt=malloc(MAXRES-100); int tn=fread(txt,1,MAXRES-100,f); fclose(f);
        /* escape */
        n += snprintf(out+n,sizeof(out)-n,"\"");
        for(int i=0;i<tn;i++){
            char c=txt[i];
            if (c=='"'){ n += snprintf(out+n,sizeof(out)-n,"\\\""); }
            else if (c=='\\'){ n += snprintf(out+n,sizeof(out)-n,"\\\\"); }
            else if (c=='\n'){ n += snprintf(out+n,sizeof(out)-n,"\\n"); }
            else if (c=='\r'||c=='\t') continue;
            else out[n++]=c;
        }
        out[n++]='"'; out[n]=0; free(txt);
    }
    strcat(out,"}");
    http_json(fd,out);
}

/* ---- cleaner.log tail ---- */
static void api_log(int fd, const char *q){
    char p[600];
    if(q && strstr(q,"type=daemon")) snprintf(p,sizeof(p),"%s/daemon.log",ADR);
    else snprintf(p,sizeof(p),"%s/cleaner.log",ADR);
    (void)q;
    FILE *f=fopen(p,"rb"); if(!f){ http_json(fd,"{\"log\":\"\"}"); return; }
    fseek(f,0,SEEK_END); long sz=ftell(f);
    /* ponytail: 日志接口只回尾部 60KB（无分页）。天花板：大日志无法回溯早期。升级路径：/api/log?tail=N    long off = sz>60000 ? sz-60000 : 0;lines 分页。 */
    long off = sz>60000 ? sz-60000 : 0;
    fseek(f,off,SEEK_SET);
    char *txt=malloc(70000); long n=fread(txt,1,70000-1,f); fclose(f); txt[n]=0;
    char *out=malloc(140000); snprintf(out,140000,"{\"log\":\"");
    int olen = strlen(out);
    for(long i=0;i<n;i++){
        char c=txt[i];
        if(c=='\n'){ out[olen++]='\\'; out[olen++]='n'; }
        else if(c=='"'){ out[olen++]='\\'; out[olen++]='"'; }
        else if(c=='\\'){ out[olen++]='\\'; out[olen++]='\\'; }
        else out[olen++]=c;
        if(olen>120000) break;
    }
    out[olen++]='"'; out[olen++]='}'; out[olen]=0;
    http_json(fd,out); free(txt); free(out);
}

/* ponytail: AI 响应用 strstr 粗解析 content（不引 JSON 库）。天花板：响应含嵌套转义/非标准格式时解析可能不准。升级路径：引入最小 JSON 解析器或捆绑 cJSON。 */
/* ---- AI: fork bundled curl, 15s timeout, POST /chat/completions ---- */
static void api_ai(int fd, const char* body){
    log_msg("INFO","AI request");
    int now=(int)time(NULL); if(now-ai_last<30){ http_err(fd,429,"{\"e\":\"AI 请求频率过高，请30秒后再试\"}"); return; } ai_last=now;
    char base[512]="", key[512]="", model[128]="";
    char p[600]; snprintf(p,sizeof(p),"%s/config.conf",ADR);
    FILE *f=fopen(p,"r");
    if (f){
        char line[600];
        while(fgets(line,sizeof(line),f)){
            char *v;
            if(!strncmp(line,"ai_base=",8)){ v=line+8; v[strcspn(v,"\r\n")]=0; snprintf(base,sizeof(base),"%s",v); }
            else if(!strncmp(line,"ai_key=",7)){ v=line+7; v[strcspn(v,"\r\n")]=0; snprintf(key,sizeof(key),"%s",v); }
            else if(!strncmp(line,"ai_model=",9)){ v=line+9; v[strcspn(v,"\r\n")]=0; snprintf(model,sizeof(model),"%s",v); }
        }
        fclose(f);
    }
    if (!*base || !*model){ http_err(fd,400,"{\"e\":\"AI 未配置：请在设置页填写 API 地址/模型\"}"); return; }
    /* load scan.json as aggregate context */
    char sc[700]; snprintf(sc,sizeof(sc),"%s/scan.json",ADR);
    char agg[4096]="{}";
    f=fopen(sc,"r"); if(f){ int n=fread(agg,1,sizeof(agg)-1,f); fclose(f); agg[n]=0; }
    /* build payload */
    char prompt[2048];
    snprintf(prompt,sizeof(prompt),
        "你是手机存储清理助手。设备体检数据：%s。请用中文给出建议：\\n"
        "1)按优先级列出建议清理的前3类及理由（删除影响）\\n"
        "2)指出红线数据（聊天媒体/下载文件）不要动\\n"
        "3)预计可释放空间。不要建议删除系统关键文件。", agg);
    char url[700];
    /* base 可能是 https://host/v1 或完整 .../chat/completions */
    if (strstr(base,"/chat/completions")) snprintf(url,sizeof(url),"%s",base);
    else snprintf(url,sizeof(url),"%s/chat/completions", base);
    char req[4096];
    snprintf(req,sizeof(req),
        "{\"model\":\"%s\",\"messages\":[{\"role\":\"user\",\"content\":\"%s\"}],\"max_tokens\":600}",
        model, prompt);
    /* temp file payload to avoid argv limits */
    char rq[600]; snprintf(rq,sizeof(rq),"%s/.ai_req.json",ADR);
    f=fopen(rq,"w"); if(f){ fwrite(req,1,strlen(req),f); fclose(f); }
    char out[600]; snprintf(out,sizeof(out),"%s/.ai_resp.json",ADR);
    char curlbin[600]; snprintf(curlbin,sizeof(curlbin),"%s/bin/curl",MODDIR);
    char cmdline[2048];
    snprintf(cmdline,sizeof(cmdline),
        "\"%s\" -s -m 15 -X POST \"%s\" -H \"Content-Type: application/json\" -H \"Authorization: Bearer %s\" -d @%s -o %s",
        curlbin, url, key, rq, out);
    /* run synchronously with 15+s timeout */
    char *shcmd = malloc(2300);
    /* ponytail: AI 响应截断 4000B（防超大回包）。天花板：长回答被截断。升级路径：流式读取或增大上限。 */
    snprintf(shcmd,2300,"%s; echo __JC__$?; head -c 4000 %s 2>/dev/null", cmdline, out);
    if(getenv("JC_DEBUG")){ FILE*dbg=fopen("/tmp/jc/run/cmd.dbg","w"); fprintf(dbg,"%s",shcmd); fclose(dbg); }
    int pipefd[2]; pipe(pipefd);
    pid_t pid=fork();
    if(pid==0){ dup2(pipefd[1],1); close(pipefd[0]);
        execl(SH,SH,"-c",shcmd,(char*)NULL);
        int er=errno; FILE*e=fopen("/tmp/jc/run/exec.err","w"); fprintf(e,"execfail errno=%d(%s)",er,strerror(er)); fclose(e);
        _exit(127); }
    close(pipefd[1]);
    struct timeval tv={18,0}; setsockopt(pipefd[0],SOL_SOCKET,SO_RCVTIMEO,&tv,sizeof(tv));
    char res[8192]=""; long rdtotal=0; char rdlog[256]; int rdlogn=0;
    for(;;){
        ssize_t rd=read(pipefd[0],res+rdtotal,sizeof(res)-1-rdtotal);
        rdlogn+=snprintf(rdlog+rdlogn,sizeof(rdlog)-rdlogn,"rd=%zd ",rd);
        if(rd<=0) break;
        rdtotal+=rd;
        if(rdtotal>7000) break;
    }
    res[rdtotal]=0;
    if(getenv("JC_DEBUG")){ FILE*dbg=fopen("/tmp/jc/run/ai.dbg","w"); fprintf(dbg,"%s\n--res--\n%.2000s",rdlog,res); fclose(dbg); }
    close(pipefd[0]);
    int st; 
    for(int i=0;i<30;i++){ if(waitpid(pid,&st,WNOHANG)==pid) break; usleep(500000); }
    kill(pid,SIGKILL); waitpid(pid,&st,0);
    /* parse choices[0].message.content */
    char *ch = strstr(res,"\"content\"");
    char text[7000]="";
    if (ch){
        char *c2 = strchr(ch,':');
        if (c2){ c2++; while(*c2==' '||*c2=='\t') c2++;
            if (*c2=='"'){ c2++; char *e=strchr(c2,'"');
                if (e){ int len=(int)(e-c2); if(len>6800)len=6800; memcpy(text,c2,len); text[len]=0; }
            }
        }
    } 
    if (!*text){
        /* fallback: raw first line */
        char *nl=strchr(res,'\n'); int len = nl?(int)(nl-res):(int)strlen(res);
        if(len>6800)len=6800; memcpy(text,res,len); text[len]=0;
    }
    unlink(rq); unlink(out);
    /* JSON-escape */
    char *js=malloc(14000); snprintf(js,14000,"{\"ai\":\"");
    int o=7;
    for(char *c=text;*c && o<13600;c++){
        if(*c=='"'){js[o++]='\\';js[o++]='"';}
        else if(*c=='\n'){js[o++]='\\';js[o++]='n';}
        else if(*c=='\\'){js[o++]='\\';js[o++]='\\';}
        else js[o++]=*c;
    }
    js[o++]='"'; js[o++]='}'; js[o]=0;
    http_json(fd,js); free(js); free(shcmd);
}


/* whitelist check: return 1 if path is protected (prefix + path-boundary) */
static int wl_block(const char *path){
    char p[600]; snprintf(p,sizeof(p),"%s/rules/whitelist.list",ADR);
    FILE *f=fopen(p,"r"); if(!f) return 0;
    char line[700]; int block=0;
    while(fgets(line,sizeof(line),f)){
        line[strcspn(line,"\r\n")]=0;
        if(!*line||line[0]=='#') continue;
        size_t wl=strlen(line);
        if(!strncasecmp(path,line,wl)&&(path[wl]==0||path[wl]=='/')){ block=1; break; }
    }
    fclose(f); return block;
}
/* ---- check path existence: POST {"paths":["/a","/b"]} → {"exists":{"/a":1,"/b":0}} ---- */
static void api_check(int fd, const char* body){
    /* 提取所有 / 开头到 " , ] } 结束的路径段，检查存在性 */
    if(!body||!*body){ http_err(fd,400,"{\"e\":\"empty\"}"); return; }
    char out[4096]; int o=snprintf(out,sizeof(out),"{\"exists\":{");
    int first=1; const char *p=body;
    while(*p && o < 3800){
        if(*p=='/'){
            const char *e=p; while(*e && *e!='"' && *e!=',' && *e!=']' && *e!='}') e++;
            int len=(int)(e-p); if(len>500)len=500;
            char buf[512]; memcpy(buf,p,len); buf[len]=0;
            int ex=access(buf,F_OK)==0;
            o += snprintf(out+o,sizeof(out)-o,"%s\"%.*s\":%d",first?"":",",len,p,ex);
            first=0; p=e;
        } else p++;
    }
    o += snprintf(out+o,sizeof(out)-o,"}}");
    http_json(fd,out);
}
/* ---- delete user-confirmed big files (argv-safe rm, prefix guard) ---- */
static void api_delbig(int fd, const char* body){
    if(!body||!*body){ http_err(fd,400,"{\"e\":\"empty\"}"); return; }
    char tmp[4096]; snprintf(tmp,sizeof(tmp),"%s",body);
    int deleted=0;
    char* tok=strtok(tmp,",");
    while(tok){
        size_t l=strlen(tok);
        while(l&&(tok[l-1]==' '||tok[l-1]=='"')) tok[--l]=0;
        if((!strncmp(tok,"/sdcard/",8)||!strncmp(tok,"/storage/emulated/",18))
           && !strstr(tok,"/../") && !strstr(tok,"/..") && !strcmp(tok,"..")
           && !strchr(tok,';')&&!strchr(tok,'&')&&!strchr(tok,'|')&&!strchr(tok,'$')&&!strchr(tok,'`')&&!strchr(tok,'*')&&!strchr(tok,'?')
           && !wl_block(tok)){
            pid_t c=fork();
            if(c==0){ execl("/bin/rm","rm","-rf",tok,(char*)NULL); execl("/system/bin/rm","rm","-rf",tok,(char*)NULL); _exit(127); }
            int st=0; waitpid(c,&st,0);
            if(WIFEXITED(st)&&WEXITSTATUS(st)==0) deleted++;
        }
        tok=strtok(NULL,",");
    }
    char out[96]; snprintf(out,sizeof(out),"{\"deleted\":%d}",deleted);
    http_json(fd,out);
}
/* ---- big file archive: mv to /sdcard/下载/大文件 (safe) ---- */
static void api_bigmove(int fd, const char* body){
    if(!body||!*body){ http_err(fd,400,"{\"e\":\"empty\"}"); return; }
    char tmp[4096]; snprintf(tmp,sizeof(tmp),"%s",body);
    int moved=0;
    char* tok=strtok(tmp,",");
    while(tok){
        size_t l=strlen(tok);
        while(l&&(tok[l-1]==' '||tok[l-1]=='"')) tok[--l]=0;
        if((!strncmp(tok,"/sdcard/",8)||!strncmp(tok,"/storage/emulated/",18))
           && !strstr(tok,"/../") && !strcmp(tok,"..")
           && !strchr(tok,';')&&!strchr(tok,'&')&&!strchr(tok,'|')&&!strchr(tok,'$')&&!strchr(tok,'`')&&!strchr(tok,'*')&&!strchr(tok,'?')){
            char cmd[1024];
            snprintf(cmd,sizeof(cmd),"mkdir -p '/sdcard/下载/大文件' 2>/dev/null; mv -f '%s' '/sdcard/下载/大文件/' 2>/dev/null",tok);
            int rc=system(cmd);
            if(rc==0) moved++;
        }
        tok=strtok(NULL,",");
    }
    char out[96]; snprintf(out,sizeof(out),"{\"moved\":%d}",moved);
    http_json(fd,out);
}

/* ---- progress ---- */
static void api_progress(int fd){
    char out[256];
    pthread_mutex_lock(&task_mu);
    snprintf(out,sizeof(out),"{\"running\":%d,\"pct\":%d,\"msg\":\"%s\"}", task.running, task.pct, task.msg);
    pthread_mutex_unlock(&task_mu);
    http_json(fd,out);
}

/* ---- status ---- */
/* 同步执行 cleaner.sh 命令并返回 stdout（设置环境变量） */
static char* run_sync(const char* args){
    static char cmd[1024]; snprintf(cmd,sizeof(cmd),"JC_ADR='%s' RULES='%s/rules' CFG='%s/config.conf' SCAN='%s/scan.json' LOG='%s/cleaner.log' sh '%s/cleaner.sh' %s 2>/dev/null", ADR, ADR, ADR, ADR, ADR, MODDIR, args);
    FILE *f=popen(cmd,"r"); if(!f) return NULL;
    char *out=malloc(8192); int n=fread(out,1,8191,f); out[n]=0; pclose(f);
    return out;
}

static void api_status(int fd){
    char out[1024];
    char p[600]; snprintf(p,sizeof(p),"%s/cleaner.sh",MODDIR);
    pthread_mutex_lock(&task_mu);
    int busy = task.running;
    pthread_mutex_unlock(&task_mu);
    /* 累计统计: stats.total 存 "del_count freed_kb" */
    long sdel=0, skb=0;
    { char sp[512]; snprintf(sp,sizeof(sp),"%s/stats.total",ADR);
      FILE *sf=fopen(sp,"r");
      if(sf){ char b[128]; if(fgets(b,sizeof(b),sf)) sscanf(b,"%ld %ld",&sdel,&skb); fclose(sf); } }
    snprintf(out,sizeof(out),
        "{\"daemon\":1,\"busy\":%d,\"port\":%d,\"start\":\"%d\",\"stats\":{\"del\":%ld,\"kb\":%ld}}", busy, PORT, (int)time(NULL), sdel, skb);
    http_json(fd,out);
}

/* ---- tasks (timer) management: tasks.conf lines: enable=1,every=12h|daily=03:00,cats=cache,social,sqlite ---- */
static void api_tasks(int fd, const char *method, const char *body){
    char p[600]; snprintf(p,sizeof(p),"%s/tasks.conf",ADR);
    if(!strcmp(method,"POST")){
        if(!body||!*body){ http_err(fd,400,"{\"e\":\"empty\"}"); return; }
        char tmp[600]; snprintf(tmp,sizeof(tmp),"%s/tasks.conf.tmp",ADR);
        FILE*f=fopen(tmp,"w"); if(!f){ http_err(fd,500,"{\"e\":\"w\"}"); return; }
        fwrite(body,1,strlen(body),f); fclose(f);
        rename(tmp,p); chmod(p,0600);
        http_json(fd,"{\"ok\":1}");
        return;
    }
    FILE*f=fopen(p,"r"); if(!f){ http_json(fd,"{\"tasks\":[]}"); return; }
    char *txt=malloc(16384); int n=fread(txt,1,16383,f); fclose(f); txt[n]=0;
    /* wrap as {"tasks":[...lines...]} keep simple: return raw text */
    char *out=malloc(32768); snprintf(out,32768,"{\"tasks\":\"");
    int o=9;
    for(int i=0;i<n;i++){ char c=txt[i];
        if(c=='\n'){out[o++]='\\';out[o++]='n';} else if(c=='"'){out[o++]='\\';out[o++]='"';} else out[o++]=c;
        if(o>32000)break;
    }
    out[o++]='"'; out[o++]='}'; out[o]=0;
    http_json(fd,out); free(txt); free(out);
}

/* ---- timer loop: every 60s check tasks.conf; run authorized cats when due ---- */
static int battery_charging(void){
    DIR *d=opendir("/sys/class/power_supply"); if(!d) return 0;
    struct dirent *e; int ok=0;
    while((e=readdir(d))){
        if(e->d_name[0]=='.') continue;
        char p[256]; snprintf(p,sizeof(p),"/sys/class/power_supply/%s/status",e->d_name);
        FILE *f=fopen(p,"r"); if(!f) continue;
        char s[32]; if(fgets(s,sizeof(s),f)){ if(!strncmp(s,"Charging",8)||!strncmp(s,"Full",4)) ok=1; }
        fclose(f);
    }
    closedir(d); return ok;
}
static int wifi_up(void){
    FILE *f=fopen("/sys/class/net/wlan0/operstate","r"); if(!f) return 0;
    char s[16]; int ok=0; if(fgets(s,sizeof(s),f)){ if(!strncmp(s,"up",2)) ok=1; } fclose(f); return ok;
}
static int idle_low(void){
    FILE *f=fopen("/proc/loadavg","r"); if(!f) return 1;
    float l=99; fscanf(f,"%f",&l); fclose(f); return l<1.5;
}
static void timer_loop(void){
    char p[600]; snprintf(p,sizeof(p),"%s/tasks.conf",ADR);
    struct stat stt;
    char stamp[600]; snprintf(stamp,sizeof(stamp),"%s/.task_stamp",ADR);
    while(1){
        sleep(60);
        if(!stat(p,&stt)){
            /* read tasks */
            FILE *f=fopen(p,"r"); if(!f) continue;
            int run=0;
            char line[512];
            while(fgets(line,sizeof(line),f)){
                line[strcspn(line,"\r\n")]=0;
                if(!strstr(line,"enable=1")) continue;
                char every[32]="", daily[16]="", cats[128]="";
                char *tok=strtok(line,",");
                while(tok){ 
                    if(!strncmp(tok,"every=",6)) snprintf(every,sizeof(every),"%s",tok+6);
                    else if(!strncmp(tok,"daily=",6)) snprintf(daily,sizeof(daily),"%s",tok+6);
                    else if(!strncmp(tok,"cats=",5)) snprintf(cats,sizeof(cats),"%s",tok+5);
                    tok=strtok(NULL,",");
                }
                if(!*cats) continue;
                time_t now=time(NULL); struct tm *tmv=localtime(&now);
                /* daily: match HH:MM */
                if(*daily){
                    char hhmm[8]; snprintf(hhmm,sizeof(hhmm),"%02d:%02d",tmv->tm_hour,tmv->tm_min);
                    /* 防抖：同一 HH:MM 只触发一次（记录上次触发时间） */
                    char lastd[600]; snprintf(lastd,sizeof(lastd),"%s/.daily_stamp",ADR);
                    int lastm=-1;
                    FILE *lf=fopen(lastd,"r"); if(lf){ fscanf(lf,"%d",&lastm); fclose(lf); }
                    int curm=tmv->tm_hour*60+tmv->tm_min;
                    if(strcmp(hhmm,daily)==0 && curm!=lastm){ run=1; lf=fopen(lastd,"w"); if(lf){ fprintf(lf,"%d",curm); fclose(lf); } }
                } else {
                    /* every=12h -> interval check via stamp */
                    int h=atoi(every); if(h<=0) h=12;
                    if(access(stamp,F_OK)!=0) {
                        FILE *sf=fopen(stamp,"w"); if(sf){fprintf(sf,"%ld",(long)now); fclose(sf);} run=1;
                    } else {
                        FILE *sf=fopen(stamp,"r"); long last=0; if(sf){ fscanf(sf,"%ld",&last); fclose(sf); }
                        if(now-last >= h*3600) { run=1; sf=fopen(stamp,"w"); if(sf){fprintf(sf,"%ld",(long)now); fclose(sf);} }
                    }
                }
                /* 条件：charge=1 仅充电 / wifi=1 仅WiFi / idle=1 仅空闲 */
                if(run){
                    if(strstr(line,"charge=1") && !battery_charging()) run=0;
                    if(strstr(line,"wifi=1") && !wifi_up()) run=0;
                    if(strstr(line,"idle=1") && !idle_low()) run=0;
                }
                if(run){
                    char *cmd=malloc(256);
                    snprintf(cmd,256,"clean %s",cats);
                    log_msg("INFO","timer triggered: clean %s", cats);
                    bg(cmd);
                    /* only one task per tick */
                    break;
                }
            }
            fclose(f);
        }
    }
}

/* void *api_ai_threaded(void *p); */
void *handle_threaded(void *p);
void *timer_thread_wrap(void *p);
static void api_monitor(int fd, const char *method, const char *q, const char *body);
static void *monitor_thread(void *p);

/* ponytail: API 路由用线性 if 链（<20 个可接受）。天花板：新增 API 需改此链。升级路径：函数指针路由表（API 数超过 20 时再考虑）。 */
/* ---- connection handler: parse first line + body, route ---- */
static void handle(int fd){
    char base[1024];
    char buf[32768]; int n=0;
    struct timeval tv={20,0}; setsockopt(fd,SOL_SOCKET,SO_RCVTIMEO,&tv,sizeof(tv));
    /* read until header end */
    int hdr_end=-1;
    while(n < (int)sizeof(buf)-1){
        int r=read(fd,buf+n,256);
        if(r<=0) break;
        n+=r; buf[n]=0;
        char *he=strstr(buf,"\r\n\r\n");
        if(he){ hdr_end=(int)(he-buf)+4; break; }
    }
    if(hdr_end<0){ close(fd); return; }
    if(hdr_end>=4) buf[hdr_end-4]=0;   /* 终止 header，保留 body */
    char *body_start = buf+hdr_end;
    /* parse request line INTO COPY (never corrupt shared buf:  would kill later strstr) */
    char *sp1=strchr(buf,' '); if(!sp1){ close(fd); return; }
    char method[8]; int ml=(int)(sp1-buf); if(ml>6)ml=6; memcpy(method,buf,ml); method[ml]=0;
    char *sp2=strchr(sp1+1,' '); if(!sp2){ close(fd); return; }
    int plen=(int)(sp2-sp1-1); if(plen>1023)plen=1023;
    memcpy(base,sp1+1,plen); base[plen]=0;   /* path+query copy */
    /* body length */
    long cl=0;
    char *clh=strstr(buf,"Content-Length:");
    if(clh){ cl=atol(clh+15); }
    /* read remainder of body */
    long have = (long)(n - hdr_end);
    while(have<cl && n < (int)sizeof(buf)-1){
        int r=read(fd,buf+n,sizeof(buf)-1-n);
        if(r<=0)break; n+=r; have = (long)(n - hdr_end);
    }
    if(getenv("JC_DEBUG")){ FILE*dbg=fopen("/tmp/jc/run/req.dbg","w");
        fprintf(dbg,"n=%d hdr_end=%d cl=%ld have=%ld\n",n,hdr_end,cl,have);
        fwrite(buf,1,n>80?80:n,dbg); fclose(dbg); }
    char *body = body_start;
    if(have>cl) body[cl]=0;
    char *q=strchr(base,'?');
    /* route */
    if(!strncmp(base,"/api/stats-history",18)){ char *out=run_sync("history"); http_json(fd,out?out:"{\"history\":[]}"); free(out); }
    if(!strncmp(base,"/api/status",11)) api_status(fd);
    else if(!strncmp(base,"/api/config",11)) api_config(fd, method, (char*)body);
    else if(!strncmp(base,"/api/rules",10)) api_rules(fd, method, q?q:"", (char*)body);
    else if(!strncmp(base,"/api/log",8)) api_log(fd, q?q:"");
    else if(!strncmp(base,"/api/progress",13)) api_progress(fd);
    else if(!strncmp(base,"/api/tasks",10)) api_tasks(fd, method, (char*)body);
    else if(!strncmp(base,"/api/cleanapp",13) && !strcmp(method,"POST")){
        char pkg[128]; snprintf(pkg,sizeof(pkg),"%s",body?body:"");
        size_t pl=strlen(pkg); while(pl&&(pkg[pl-1]==' '||pkg[pl-1]=='"')) pkg[--pl]=0;
        if(!*pkg||strchr(pkg,'/')||strstr(pkg,"..")){ http_err(fd,400,"{\"e\":\"bad\"}"); return; }
        char cmd[256]; snprintf(cmd,sizeof(cmd),"cleanapp %s",pkg);
        char *out=run_sync(cmd); http_json(fd,out?out:"{\"ok\":0}"); free(out);
    }
    else if(!strncmp(base,"/api/clean",10) && !strcmp(method,"POST")){
        char cats[128]="all", force[8]="";
        jget(body,"cats",cats,sizeof(cats));
        int hasforce = (q?strstr(q,"force"):NULL)!=0 || jget(body,"force",force,sizeof(force))==0;
        char *cmd=malloc(256);
        snprintf(cmd,256,"clean %s%s",cats,hasforce?" force":"");
        bg(cmd);
        http_json(fd,"{\"ok\":1,\"started\":1}");
    }
    else if(!strncmp(base,"/api/scan",9)){
        if(!strcmp(method,"POST")){ char *cmd=strdup(body&&strstr(body,"force")?"scan force":"scan"); bg(cmd); http_json(fd,"{\"ok\":1,\"scanning\":1}"); }
        else { /* serve cached scan.json */ char p[600]; snprintf(p,sizeof(p),"%s/scan.json",ADR); FILE*f=fopen(p,"r");
            if(f){ char *t=malloc(MAXRES); int sz=fread(t,1,MAXRES-1,f); fclose(f); t[sz]=0; http_json(fd,t); free(t); }
            else http_err(fd,404,"{\"e\":\"no scan yet\"}");
        }
    }
    else if(!strncmp(base,"/api/ai",7)){
        /* run in thread to not block server */
        api_ai(fd,(char*)body); close(fd); return;
    }
    else if(!strncmp(base,"/api/check",10) && !strcmp(method,"POST")) api_check(fd,(char*)body);
    else if(!strncmp(base,"/api/delbig",11) && !strcmp(method,"POST")) api_delbig(fd,(char*)body);
    else if(!strncmp(base,"/api/bigmove",12) && !strcmp(method,"POST")) api_bigmove(fd,(char*)body);
    else if(!strncmp(base,"/api/classify-preview",21)){
        char p[640]; snprintf(p,sizeof(p),"%s/.classify.preview.json",ADR);
        FILE *f=fopen(p,"r"); if(!f){ http_json(fd,"{\"files\":[]}"); return; }
        char *txt=malloc(32768); int n=fread(txt,1,32767,f); fclose(f); txt[n]=0;
        http_json(fd,txt); free(txt);
    }
    else if(!strncmp(base,"/api/classify",13)){ char *c=strdup(body&&strstr(body,"preview")?"classify preview":"classify"); bg(c); http_json(fd,"{\"ok\":1}"); }
    else if(!strncmp(base,"/api/duplicate-preview",22)){
        char p[640]; snprintf(p,sizeof(p),"%s/.dup.preview.json",ADR);
        FILE *f=fopen(p,"r"); if(!f){ http_json(fd,"{\"files\":[]}"); return; }
        char *txt=malloc(32768); int n=fread(txt,1,32767,f); fclose(f); txt[n]=0;
        http_json(fd,txt); free(txt);
    }
    else if(!strncmp(base,"/api/duplicate",14)){
        char *cmd=0;
        if(body&&strstr(body,"preview")) cmd=strdup("duplicate preview");
        else if(body&&strstr(body,"keep")){
            const char *kp=strstr(body,"\"keep\":\"");
            if(kp){ kp+=8; const char *ke=strchr(kp,'"'); char kpath[600]=""; if(ke){ int kl=(int)(ke-kp); if(kl>590)kl=590; snprintf(kpath,sizeof(kpath),"%.*s",kl,kp); }
                char tmp[700]; snprintf(tmp,sizeof(tmp),"duplicate keep %s",kpath); cmd=strdup(tmp); }
            else cmd=strdup("duplicate delete");
        }
        else if(body&&strstr(body,"delete")){
            const char *kp=strstr(body,"\"keep\":\"");
            if(kp){ kp+=8; const char *ke=strchr(kp,'"'); char kpath[600]=""; if(ke){ int kl=(int)(ke-kp); if(kl>590)kl=590; snprintf(kpath,sizeof(kpath),"%.*s",kl,kp); } char tmp[700]; snprintf(tmp,sizeof(tmp),"duplicate delete %s",kpath); cmd=strdup(tmp); }
            else cmd=strdup("duplicate delete");
        }
        else cmd=strdup("duplicate");
        bg(cmd); http_json(fd,"{\"ok\":1}");
    }
    else if(!strncmp(base,"/api/fstrim",11)){ char *c=strdup("fstrim"); bg(c); http_json(fd,"{\"ok\":1}"); }
    else if(!strncmp(base,"/api/rescan",11)){ char *c=strdup("rescan"); bg(c); http_json(fd,"{\"ok\":1}"); }
    else if(!strncmp(base,"/api/monitor",12)) api_monitor(fd, method, q?q:"", (char*)body);
    else serve_file(fd, base);
    close(fd);
}
/* wrapper for AI in its own thread */
/* ai 同步调用，不再线程化（body 支持多轮对话） */

int main(int argc, char**argv){
    for(int i=1;i<argc;i++){
        if(!strcmp(argv[i],"-d")&&i+1<argc) snprintf(ADR,sizeof(ADR),"%s",argv[++i]);
        else if(!strcmp(argv[i],"-m")&&i+1<argc) snprintf(MODDIR,sizeof(MODDIR),"%s",argv[++i]);
        else if(!strcmp(argv[i],"-sh")&&i+1<argc) snprintf(SH,sizeof(SH),"%s",argv[++i]);
    }
    signal(SIGPIPE,SIG_IGN);
    signal(SIGTERM,cleanup_exit); signal(SIGINT,cleanup_exit);
    log_msg("INFO","daemon starting (ADR=%s MOD=%s port=%d)", ADR, MODDIR, PORT);
    /* ensure runtime dirs */
    char p[600]; snprintf(p,sizeof(p),"%s/rules",ADR); mkdir(p,0755);
    pthread_t timer; if(pthread_create(&timer,NULL,(void*(*)(void*))timer_thread_wrap,NULL)!=0){ log_msg("ERR","pthread_create timer failed"); }
    pthread_t mth; if(pthread_create(&mth,NULL,monitor_thread,NULL)!=0){ log_msg("ERR","pthread_create monitor failed"); } else pthread_detach(mth);
    int lfd=socket(AF_INET,SOCK_STREAM,0);
    int one=1; setsockopt(lfd,SOL_SOCKET,SO_REUSEADDR,&one,sizeof(one));
    struct sockaddr_in sa; memset(&sa,0,sizeof(sa));
    sa.sin_family=AF_INET; sa.sin_addr.s_addr=htonl(INADDR_LOOPBACK); sa.sin_port=htons(PORT);
    if(bind(lfd,(struct sockaddr*)&sa,sizeof(sa))<0){
        log_msg("ERR","bind port %d failed: %s", PORT, strerror(errno));
        sleep(2);
        if(bind(lfd,(struct sockaddr*)&sa,sizeof(sa))<0){
            log_msg("ERR","bind retry failed, exiting: %s", strerror(errno));
            _exit(1);
        }
    }
    if(listen(lfd,16)<0){ log_msg("ERR","listen failed: %s", strerror(errno)); _exit(1); }
    log_msg("INFO","daemon running on 127.0.0.1:%d", PORT);
    for(;;){
        int cfd=accept(lfd,NULL,NULL);
        if(cfd<0){ usleep(200000); continue; }
        pthread_t ht;
        if(pthread_create(&ht,NULL,(void*(*)(void*))handle_threaded,(void*)(long)cfd)!=0){
            log_msg("WARN","pthread_create handler failed: %s", strerror(errno)); close(cfd); continue;
        }
        pthread_detach(ht);
    }
}
static void api_monitor(int fd, const char *method, const char *q, const char *body){
    (void)q;
    char p[600]; snprintf(p,sizeof(p),"%s/config.conf",ADR);
    /* GET */
    if(!strcmp(method,"GET")){
        int on=0; char dirs[512]="",logbuf[600]="";
        FILE *f=fopen(p,"r");
        if(f){ char line[600];
            while(fgets(line,sizeof(line),f)){
                if(!strncmp(line,"monitor=1",9)) on=1;
                if(!strncmp(line,"monitor_dirs=",13)){ char *v=line+13; v[strcspn(v,"\r\n")]=0; if(strlen(v)){ if(dirs[0]) strncat(dirs,",",sizeof(dirs)-strlen(dirs)-1); strncat(dirs,v,sizeof(dirs)-strlen(dirs)-1); } }
            } fclose(f);
        }
        char ml[600]; snprintf(ml,sizeof(ml),"%s/monitor.log",ADR);
        FILE *lf=fopen(ml,"r"); if(lf){ int n=fread(logbuf,1,sizeof(logbuf)-1,lf); fclose(lf); logbuf[n]=0; }
        char out[1300]; snprintf(out,sizeof(out),"{\"on\":%d,\"dirs\":\"%s\",\"log\":\"%s\"}",on,dirs,logbuf);
        http_json(fd,out); return;
    }
    /* POST */
    if(!strcmp(method,"POST")&&body&&*body){
        char dirs[1024]="";
        FILE *rf=fopen(p,"r");
        if(rf){ char line[600]; while(fgets(line,sizeof(line),rf)){
            if(!strncmp(line,"monitor_dirs=",13)){ char *v=line+13; v[strcspn(v,"\r\n")]=0; snprintf(dirs,sizeof(dirs),"%s",v); }
        } fclose(rf); }
        /* add */
        const char *ad=strstr(body,"\"add\":\"");
        if(ad){ ad+=7; const char *de=strchr(ad,'"'); char path[300]={0}; if(de&&de-ad<300){ memcpy(path,ad,de-ad);
            if(strncmp(path,"/sdcard/",8)&&strncmp(path,"/storage/emulated/",18)){ http_err(fd,400,"{\"e\":\"bad\"}"); return; }
            if(!strstr(dirs,path)){ if(dirs[0]) strncat(dirs,",",sizeof(dirs)-strlen(dirs)-1); strncat(dirs,path,sizeof(dirs)-strlen(dirs)-1); }
        }}
        /* remove */
        const char *rm=strstr(body,"\"remove\":\"");
        if(rm&&dirs[0]){ rm+=10; const char *re=strchr(rm,'"'); char path[300]={0}; if(re&&re-rm<300){ memcpy(path,rm,re-rm);
            char nd[1024]=""; const char *st=dirs; while(*st){ const char *c=strchr(st,','); int len=c?(int)(c-st):(int)strlen(st);
                char seg[512]={0}; memcpy(seg,st,len);
                if(strcmp(seg,path)){ if(nd[0]) strncat(nd,",",sizeof(nd)-strlen(nd)-1); strncat(nd,seg,sizeof(nd)-strlen(nd)-1); }
                if(!c)break; st=c+1; }
            snprintf(dirs,sizeof(dirs),"%s",nd);
        }}
        int set_mon=-1;
        if(strstr(body,"\"on\":1")) set_mon=1; else if(strstr(body,"\"on\":0")) set_mon=0;
        /* fgets 逐行重建 */
        char tmp[600]; snprintf(tmp,sizeof(tmp),"%s.tmp",p);
        FILE *wi=fopen(tmp,"w"); if(!wi){ http_err(fd,500,"{\"e\":\"write\"}"); return; }
        FILE *rd=fopen(p,"r");
        int wrote_mon=0, wrote_dirs=0;
        if(rd){ char ln[600]; while(fgets(ln,sizeof(ln),rd)){
            if(!strncmp(ln,"monitor=",8)){ wrote_mon=1; if(set_mon>=0) fprintf(wi,"monitor=%d\n",set_mon); else fputs(ln,wi); }
            else if(!strncmp(ln,"monitor_dirs=",13)){ wrote_dirs=1; if(dirs[0]) fprintf(wi,"monitor_dirs=%s\n",dirs); else fputs(ln,wi); }
            else fputs(ln,wi);
        } fclose(rd); }
        if(!wrote_mon && set_mon>=0) fprintf(wi,"monitor=%d\n",set_mon);
        if(!wrote_dirs && (strstr(body,"add")||strstr(body,"remove"))) fprintf(wi,"monitor_dirs=%s\n",dirs[0]?dirs:"");
        fclose(wi);
        if(rename(tmp,p)!=0){ http_err(fd,500,"{\"e\":\"rename\"}"); return; }
        chmod(p,0600);
        http_json(fd,"{\"ok\":1}"); return;
    }
    http_err(fd,400,"{\"e\":\"method\"}");
}

/* monitor 轮询线程（30s 扫新文件→classify） */
static void *monitor_thread(void *p){
    (void)p;
    char marker[600]; snprintf(marker,sizeof(marker),"%s/.mon_marker",ADR);
    for(;;){
        char cp[600]; snprintf(cp,sizeof(cp),"%s/config.conf",ADR);
        FILE *cf=fopen(cp,"r"); int on=0; char dirs[16][512]; int nd=0;
        if(cf){ char line[600]; while(fgets(line,sizeof(line),cf)){
            if(!strncmp(line,"monitor=1",9)) on=1;
            if(!strncmp(line,"monitor_dirs=",13)){ char *v=line+13; v[strcspn(v,"\r\n")]=0; if(nd<16) snprintf(dirs[nd++],512,"%s",v); }
        } fclose(cf); }
        if(!on||nd==0){ sleep(30); continue; }
        for(int i=0;i<nd;i++){ char cmd[1100]; snprintf(cmd,sizeof(cmd),"find %s -maxdepth 1 -type f -newer %s 2>/dev/null | head -20",dirs[i],marker);
            char tmp[64]; snprintf(tmp,sizeof(tmp),"/tmp/jcmon%d",getpid()); char full[1100]; snprintf(full,sizeof(full),"%s > %s 2>/dev/null",cmd,tmp);
            system(full); FILE *rf=fopen(tmp,"r"); if(!rf) continue;
            char line[1024]; while(fgets(line,sizeof(line),rf)){ line[strcspn(line,"\r\n")]=0; if(!*line) continue;
                const char *bn=strrchr(line,'/'); bn=bn?bn+1:line; if(bn[0]=='.'||strstr(bn,".part")||strstr(bn,".tmp")) continue;
                char cl[1100]; snprintf(cl,sizeof(cl),"MON_FILE='%s' %s classify",line,SH);
                log_msg("INFO","monitor: classify %s",line); system(cl);
                char ml[600]; snprintf(ml,sizeof(ml),"%s/monitor.log",ADR); FILE *lf=fopen(ml,"a");
                if(lf){ time_t t=time(NULL); struct tm*tm=localtime(&t); fprintf(lf,"[%04d-%02d-%02d %02d:%02d:%02d] %s\n",tm->tm_year+1900,tm->tm_mon+1,tm->tm_mday,tm->tm_hour,tm->tm_min,tm->tm_sec,line); fclose(lf); }
            } fclose(rf); unlink(tmp);
        }
        char t[700]; snprintf(t,sizeof(t),"touch %s",marker); system(t);
        sleep(30);
    }
    return NULL;
}

void *timer_thread_wrap(void*p){ (void)p; timer_loop(); return NULL; }
void *handle_threaded(void *p){ long fd=(long)p; handle((int)fd); return NULL; }

