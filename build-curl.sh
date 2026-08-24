#!/bin/bash
# JunkClean - 交叉编译静态 curl (aarch64 + OpenSSL)
# 用途: bin/curl 不入 git（二进制），服务器重建后需重新编译
# 依赖: aarch64-linux-gnu-gcc, perl, make; 外网
set -e
cd "$(dirname "$0")"
TOOLS=aarch64-linux-gnu-
DL=/tmp/jc-build
mkdir -p $DL
OSSL=openssl-3.3.2
CURL=curl-8.10.1

echo "=== 1. 下载源码 ==="
[ -f $DL/$OSSL.tar.gz ] || curl -sL -o $DL/$OSSL.tar.gz https://www.openssl.org/source/$OSSL.tar.gz
[ -f $DL/$CURL.tar.gz ] || curl -sL -o $DL/$CURL.tar.gz https://curl.se/download/$CURL.tar.gz

echo "=== 2. 编译 OpenSSL ==="
cd $DL && rm -rf $OSSL && tar xzf $OSSL.tar.gz && cd $OSSL
./Configure linux-aarch64 --prefix=$DL/openssl-out no-shared no-tests -static --cross-compile-prefix=$TOOLS >/dev/null 2>&1
make -j$(nproc) >/dev/null 2>&1 && make install_sw >/dev/null 2>&1
echo "   OpenSSL OK"

echo "=== 3. 编译 curl ==="
cd $DL && rm -rf $CURL && tar xzf $CURL.tar.gz && cd $CURL
# 必须 --without-zlib（无 aarch64 静态 zlib）
./configure --host=aarch64-linux-gnu --disable-shared --enable-static \
  LDFLAGS="-static" CFLAGS="-O2" \
  --with-ssl=$DL/openssl-out --without-zlib \
  --disable-ldap --disable-ldaps --disable-rtsp --disable-dict --disable-telnet \
  --disable-tftp --disable-pop3 --disable-imap --disable-smtp --disable-gopher \
  --disable-manual --disable-docs --without-libpsl --without-libidn2 \
  --without-brotli --without-zstd --without-libssh2 --without-nghttp2 \
  --without-ngtcp2 --without-quic >/dev/null 2>&1
# -all-static 强制完全静态（关键！普通 -static 会残留 libc 动态依赖）
make -j$(nproc) CCLD="$TOOLS gcc" LDFLAGS="-all-static -L$DL/openssl-out/lib" >/dev/null 2>&1
$TOOLS strip -s src/curl
cp src/curl $OLDPWD/bin/curl
file src/curl | grep -q static || { echo "✗ curl 非静态，失败"; exit 1; }
echo "   curl OK -> bin/curl ($(ls -la $OLDPWD/bin/curl | awk '{print $5}')B)"

echo "=== 完成 ==="
