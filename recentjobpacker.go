package dcron

import (
	"container/heap"
	"sync"
	"time"
)

type JobWithTime struct {
	JobName     string
	RunningTime time.Time
}

type JobWithTimeHeap []JobWithTime

func (jobHeap *JobWithTimeHeap) Pop() (ret interface{}) {
	n := jobHeap.Len() - 1
	ret = (*jobHeap)[n]
	(*jobHeap) = (*jobHeap)[0:n]
	return
}

func (jobHeap *JobWithTimeHeap) Push(x interface{}) {
	(*jobHeap) = append((*jobHeap), x.(JobWithTime))
}

func (jobHeap *JobWithTimeHeap) Less(i, j int) bool {
	return (*jobHeap)[i].RunningTime.Before((*jobHeap)[j].RunningTime)
}

func (jobHeap *JobWithTimeHeap) Len() int {
	return len(*jobHeap)
}

func (jobHeap *JobWithTimeHeap) Swap(i, j int) {
	t := (*jobHeap)[i]
	(*jobHeap)[i] = (*jobHeap)[j]
	(*jobHeap)[j] = t
}

func (jobHeap *JobWithTimeHeap) Index(i int) interface{} {
	return (*jobHeap)[i]
}

type RecentJobPacker struct {
	sync.Mutex
	timeWindow time.Duration
	recentJobs JobWithTimeHeap
}

func (rjp *RecentJobPacker) AddJob(jobName string, t time.Time) (err error) {
	rjp.Lock()
	defer rjp.Unlock()
	heap.Push(&rjp.recentJobs, JobWithTime{
		JobName:     jobName,
		RunningTime: t,
	})

	unRencentTime := time.Now().Add(-rjp.timeWindow)
	for rjp.recentJobs.Len() > 0 && rjp.recentJobs.Index(0).(JobWithTime).RunningTime.Before(unRencentTime) {
		heap.Pop(&rjp.recentJobs)
	}
	return
}

func (rjp *RecentJobPacker) PopAllJobs() (jobNames []string) {
	rjp.Lock()
	defer rjp.Unlock()
	jobNames = make([]string, 0)
	for rjp.recentJobs.Len() > 0 {
		jobNames = append(jobNames, heap.Pop(&rjp.recentJobs).(JobWithTime).JobName)
	}
	rjp.recentJobs = make(JobWithTimeHeap, 0)
	heap.Init(&rjp.recentJobs)
	return
}

func NewRecentJobPacker(timeWindow time.Duration) IRecentJobPacker {
	return &RecentJobPacker{
		timeWindow: timeWindow,
		recentJobs: make([]JobWithTime, 0),
	}
}
