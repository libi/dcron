# example

This [code](example.go) is an example code for dcron.

## compile

```bash
go build -o bin/example example.go
```

## run
```bash
# ./run.sh $number_of_process $number_of_cronjob
# in linux
./run.sh 5 10
```
## run 1 instance
```bash
# in linux
# ./run-instance.sh $sub_id_for_this_process $number_of_cronjob
./run-instance.sh 6 10
```

## stop all
```bash
# in linux
./killexamples.sh
```

## stop 1 instance
```bash
# in linux
# ./kill-instance.sh $sub_id
./kill-instance.sh 2
```