#!/bin/sh
cd "$(dirname "$0")"
./sshd -nohup -config conf/conf.test.yaml &