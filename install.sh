#!/bin/sh

# 根据thrift接口定义文件自动生成目标代码
thrift -out src -r --gen go weibosender.thrift

# 编译安装weibosender-server服务程序
go install weibosender-server

