package dcron

type JobWarpper struct {
	Dcron *Dcron
	Name string
	CronStr string
	Func func()
}

func(job JobWarpper)Run(){
	//如果该任务分配给了这个节点 则允许执行
	if(job.Dcron.allowThisNodeRun(job.Name)){
		job.Func()
	}
}
