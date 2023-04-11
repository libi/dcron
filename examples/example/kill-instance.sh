#!/bin/bash
kill_instance() {
  ps -ef | grep example | grep "sub_id $1" | grep -v grep | awk '{print $2}' | xargs kill -9
}

kill_instance $1