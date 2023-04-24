start_example() {
  nohup ./bin/example -sub_id $1 -jobnumber $2 &
}

start_example $1 $2