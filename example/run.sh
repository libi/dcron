#!/bin/bash

start_example() {
  nohup ./bin/example -sub_id $1 &
}


touch tmpfile
for i in $(seq 1 $1)
do
  start_example $i &
done