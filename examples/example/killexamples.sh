#!/bin/bash
ps -ef | grep example| grep -v grep | awk '{print $2}' | xargs kill -9