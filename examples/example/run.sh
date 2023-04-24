#!/bin/bash

start_example() {
  nohup ./bin/example -sub_id $1 -jobnumber $2 &
}


for i in $(seq 1 $2)
do
  touch tmpfile$i
done

for i in $(seq 1 $1)
do
  start_example $i $2 &
done