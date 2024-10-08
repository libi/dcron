package main

import (
	"fmt"
	"log"
	"os"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/alicebob/miniredis/v2"
	"github.com/dcron-contrib/commons/dlog"
	"github.com/dcron-contrib/redisdriver"
	"github.com/libi/dcron"
	"github.com/libi/dcron/cron"
	redis "github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/stretchr/testify/suite"
)

type testDcronTestSuite struct{ suite.Suite }

type TestJobWithWG struct {
	Name string
	WG   *sync.WaitGroup
	Test *testing.T
	Cnt  *atomic.Int32
}

func (job *TestJobWithWG) Run() {
	job.Test.Logf("jobName=[%s], time=%s, job rest count=%d",
		job.Name,
		time.Now().Format("15:04:05"),
		job.Cnt.Load(),
	)
	if job.Cnt.Load() == 0 {
		return
	} else {
		job.Cnt.Store(job.Cnt.Add(-1))
		if job.Cnt.Load() == 0 {
			job.WG.Done()
		}
	}
}

func (s *testDcronTestSuite) TestMultiNodes() {
	t := s.T()
	wg := &sync.WaitGroup{}
	wg.Add(3)
	testJobWGs := make([]*sync.WaitGroup, 0)
	testJobWGs = append(testJobWGs, &sync.WaitGroup{})
	testJobWGs = append(testJobWGs, &sync.WaitGroup{})
	testJobWGs = append(testJobWGs, &sync.WaitGroup{})
	testJobWGs[0].Add(1)
	testJobWGs[1].Add(1)

	testJobs := make([]*TestJobWithWG, 0)
	testJobs = append(
		testJobs,
		&TestJobWithWG{
			Name: "s1_test1",
			WG:   testJobWGs[0],
			Test: t,
			Cnt:  &atomic.Int32{},
		},
		&TestJobWithWG{
			Name: "s1_test2",
			WG:   testJobWGs[1],
			Test: t,
			Cnt:  &atomic.Int32{},
		},
		&TestJobWithWG{
			Name: "s1_test3",
			WG:   testJobWGs[2],
			Test: t,
			Cnt:  &atomic.Int32{},
		})
	testJobs[0].Cnt.Store(5)
	testJobs[1].Cnt.Store(5)

	nodeCancel := make([](chan int), 3)
	nodeCancel[0] = make(chan int, 1)
	nodeCancel[1] = make(chan int, 1)
	nodeCancel[2] = make(chan int, 1)

	// 间隔1秒启动测试节点刷新逻辑
	rds := miniredis.RunT(t)
	defer rds.Close()
	go runNode(t, wg, testJobs, nodeCancel[0], rds.Addr())
	<-time.After(time.Second)
	go runNode(t, wg, testJobs, nodeCancel[1], rds.Addr())
	<-time.After(time.Second)
	go runNode(t, wg, testJobs, nodeCancel[2], rds.Addr())

	testJobWGs[0].Wait()
	testJobWGs[1].Wait()

	close(nodeCancel[0])
	close(nodeCancel[1])
	close(nodeCancel[2])

	wg.Wait()
}

func runNode(
	t *testing.T,
	wg *sync.WaitGroup,
	testJobs []*TestJobWithWG,
	cancel chan int,
	redisAddr string,
) {
	redisCli := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	drv := redisdriver.NewDriver(redisCli)
	dcron := dcron.NewDcronWithOption(
		t.Name(),
		drv,
		dcron.WithLogger(
			dlog.DefaultPrintfLogger(
				log.New(os.Stdout, "", log.LstdFlags))))
	// 添加多个任务 启动多个节点时 任务会均匀分配给各个节点

	var err error
	for _, job := range testJobs {
		if err = dcron.AddJob(job.Name, "* * * * *", job); err != nil {
			t.Error("add job error")
		}
	}

	dcron.Start()
	//移除测试
	dcron.Remove(testJobs[2].Name)
	<-cancel
	dcron.Stop()
	wg.Done()
}

func (s *testDcronTestSuite) Test_SecondsJob() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	redisCli := redis.NewClient(&redis.Options{
		Addr: rds.Addr(),
	})
	drv := redisdriver.NewDriver(redisCli)
	dcr := dcron.NewDcronWithOption(t.Name(), drv, dcron.CronOptionSeconds())
	err := dcr.AddFunc("job1", "*/5 * * * * *", func() {
		t.Log(time.Now())
	})
	if err != nil {
		t.Error(err)
	}
	dcr.Start()
	time.Sleep(15 * time.Second)
	dcr.Stop()
}

func runSecondNode(id string, wg *sync.WaitGroup, runningTime time.Duration, t *testing.T, redisAddr string) {
	redisCli := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	drv := redisdriver.NewDriver(redisCli)
	dcr := dcron.NewDcronWithOption(t.Name(), drv,
		dcron.CronOptionSeconds(),
		dcron.WithLogger(dlog.DefaultPrintfLogger(
			log.New(os.Stdout, "["+id+"]", log.LstdFlags),
		)),
		dcron.CronOptionChain(cron.Recover(
			cron.DefaultLogger,
		)),
		dcron.WithHashReplicas(20),
		dcron.CronOptionLocation(time.Local),
	)
	var err error
	err = dcr.AddFunc("job1", "*/5 * * * * *", func() {
		t.Log(time.Now())
	})
	require.Nil(t, err)
	err = dcr.AddFunc("job2", "*/8 * * * * *", func() {
		panic("test panic")
	})
	require.Nil(t, err)
	err = dcr.AddFunc("job3", "*/2 * * * * *", func() {
		t.Log("job3:", time.Now())
	})
	require.Nil(t, err)
	dcr.Start()
	<-time.After(runningTime)
	dcr.Stop()
	wg.Done()
}

func runSecondNodeWithLogger(id string, wg *sync.WaitGroup, runningTime time.Duration, t *testing.T, redisAddr string) {
	redisCli := redis.NewClient(&redis.Options{
		Addr: redisAddr,
	})
	drv := redisdriver.NewDriver(redisCli)
	dcr := dcron.NewDcronWithOption(
		t.Name(),
		drv,
		dcron.CronOptionSeconds(),
		dcron.WithLogger(dlog.VerbosePrintfLogger(
			log.New(os.Stdout, "["+id+"]", log.LstdFlags),
		)),
		dcron.CronOptionChain(cron.Recover(
			cron.DefaultLogger,
		)),
	)
	var err error
	err = dcr.AddFunc("job1", "*/5 * * * * *", func() {
		t.Log(time.Now())
	})
	require.Nil(t, err)
	err = dcr.AddFunc("job2", "*/8 * * * * *", func() {
		panic("test panic")
	})
	require.Nil(t, err)
	err = dcr.AddFunc("job3", "*/2 * * * * *", func() {
		t.Log("job3:", time.Now())
	})
	require.Nil(t, err)
	dcr.Start()
	<-time.After(runningTime)
	dcr.Stop()
	wg.Done()
}

func (s *testDcronTestSuite) Test_SecondJobWithPanicAndMultiNodes() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	wg := &sync.WaitGroup{}
	wg.Add(5)
	go runSecondNode("1", wg, 45*time.Second, t, rds.Addr())
	go runSecondNode("2", wg, 45*time.Second, t, rds.Addr())
	go runSecondNode("3", wg, 45*time.Second, t, rds.Addr())
	go runSecondNode("4", wg, 45*time.Second, t, rds.Addr())
	go runSecondNode("5", wg, 45*time.Second, t, rds.Addr())
	wg.Wait()
}

func (s *testDcronTestSuite) Test_SecondJobWithStopAndSwapNode() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	wg := &sync.WaitGroup{}
	wg.Add(2)
	go runSecondNode("1", wg, 60*time.Second, t, rds.Addr())
	go runSecondNode("2", wg, 20*time.Second, t, rds.Addr())
	wg.Wait()
}

func (s *testDcronTestSuite) Test_WithClusterStableNodes() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	wg := &sync.WaitGroup{}
	wg.Add(5)
	runningTime := 60 * time.Second
	startFunc := func(id string, timeWindow time.Duration, t *testing.T) {
		redisCli := redis.NewClient(&redis.Options{
			Addr: rds.Addr(),
		})
		drv := redisdriver.NewDriver(redisCli)
		dcr := dcron.NewDcronWithOption(t.Name(), drv,
			dcron.CronOptionSeconds(),
			dcron.WithLogger(dlog.DefaultPrintfLogger(
				log.New(os.Stdout, "["+id+"]", log.LstdFlags)),
			),
			dcron.WithClusterStable(timeWindow),
			dcron.WithNodeUpdateDuration(timeWindow),
		)
		var err error
		err = dcr.AddFunc("job1", "*/3 * * * * *", func() {
			t.Log(time.Now())
		})
		require.Nil(t, err)
		err = dcr.AddFunc("job2", "*/8 * * * * *", func() {
			t.Logf("job2: %v", time.Now())
		})
		require.Nil(t, err)
		err = dcr.AddFunc("job3", "* * * * * *", func() {
			t.Log("job3:", time.Now())
		})
		require.Nil(t, err)
		dcr.Start()
		<-time.After(runningTime)
		dcr.Stop()
		wg.Done()
	}

	go startFunc("1", time.Second*6, t)
	go startFunc("2", time.Second*6, t)
	go startFunc("3", time.Second*6, t)
	go startFunc("4", time.Second*6, t)
	go startFunc("5", time.Second*6, t)
	wg.Wait()
}

func (s *testDcronTestSuite) Test_SecondJobLog_Issue68() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	wg := &sync.WaitGroup{}
	wg.Add(5)
	go runSecondNodeWithLogger("1", wg, 45*time.Second, t, rds.Addr())
	go runSecondNodeWithLogger("2", wg, 45*time.Second, t, rds.Addr())
	go runSecondNodeWithLogger("3", wg, 45*time.Second, t, rds.Addr())
	go runSecondNodeWithLogger("4", wg, 45*time.Second, t, rds.Addr())
	go runSecondNodeWithLogger("5", wg, 45*time.Second, t, rds.Addr())
	wg.Wait()
}

type testGetJob struct {
	Called  int
	Reached int
	Name    string
}

func (job *testGetJob) Run() {
	job.Called++
}

func (job *testGetJob) Reach() {
	job.Reached++
}

func (s *testDcronTestSuite) Test_GetJobs_ThisNodeOnlyFalse() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	redisCli := redis.NewClient(&redis.Options{
		Addr: rds.Addr(),
	})
	drv := redisdriver.NewDriver(redisCli)
	dcr := dcron.NewDcronWithOption(
		t.Name(),
		drv,
		dcron.CronOptionSeconds(),
		dcron.WithLogger(dlog.VerbosePrintfLogger(
			log.Default(),
		)),
	)
	n := 10
	for i := 0; i < n; i++ {
		assert.Nil(
			t,
			dcr.AddJob(fmt.Sprintf("job_%d", i), "* * * * * *", &testGetJob{
				Name: fmt.Sprintf("job_%d", i),
			}),
		)
	}

	jobs := dcr.GetJobs(false)
	assert.Len(t, jobs, n)
	for _, job := range jobs {
		job.Job.Run()
	}
	for _, job := range jobs {
		assert.Equal(t, job.Name, job.Job.(*testGetJob).Name)
		assert.True(t, job.Job.(*testGetJob).Called > 0)
	}
}

func (s *testDcronTestSuite) Test_GetJob_ThisNodeOnlyFalse() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	redisCli := redis.NewClient(&redis.Options{
		Addr: rds.Addr(),
	})
	drv := redisdriver.NewDriver(redisCli)
	dcr := dcron.NewDcronWithOption(
		t.Name(),
		drv,
		dcron.CronOptionSeconds(),
		dcron.WithLogger(dlog.VerbosePrintfLogger(
			log.Default(),
		)),
	)
	n := 10
	for i := 0; i < n; i++ {
		assert.Nil(
			t,
			dcr.AddJob(fmt.Sprintf("job_%d", i), "* * * * * *", &testGetJob{
				Name: fmt.Sprintf("job_%d", i),
			}),
		)
	}

	for i := 0; i < n; i++ {
		job, err := dcr.GetJob(fmt.Sprintf("job_%d", i), false)
		s.Assert().Nil(err)
		job.Job.Run()
	}

	for i := 0; i < n; i++ {
		job, err := dcr.GetJob(fmt.Sprintf("job_%d", i), false)
		s.Assert().Nil(err)
		s.Assert().Equal(1, job.Job.(*testGetJob).Called)
	}

	_, err := dcr.GetJob("xxx", false)
	s.Assert().NotNil(err)
}

func (s *testDcronTestSuite) Test_GetJobs_ThisNodeOnlyTrue() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	redisCli := redis.NewClient(&redis.Options{
		Addr: rds.Addr(),
	})
	drv := redisdriver.NewDriver(redisCli)
	dcr := dcron.NewDcronWithOption(
		t.Name(),
		drv,
		dcron.CronOptionSeconds(),
		dcron.WithLogger(dlog.VerbosePrintfLogger(
			log.Default(),
		)),
	)
	n := 10
	for i := 0; i < n; i++ {
		assert.Nil(
			t,
			dcr.AddJob(fmt.Sprintf("job_%d", i), "* * * * * *", &testGetJob{
				Name: fmt.Sprintf("job_%d", i),
			}),
		)
	}
	result := make(chan bool, 1)
	dcr.Start()
	go func() {
		// check function
		<-time.After(5 * time.Second)
		jobs := dcr.GetJobs(true)
		s.Assert().Len(jobs, n)
		result <- true
	}()
	s.Assert().True(<-result)
}

func (s *testDcronTestSuite) Test_GetJob_ThisNodeOnlyTrue() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	redisCli := redis.NewClient(&redis.Options{
		Addr: rds.Addr(),
	})
	drv := redisdriver.NewDriver(redisCli)
	dcr := dcron.NewDcronWithOption(
		t.Name(),
		drv,
		dcron.CronOptionSeconds(),
		dcron.WithLogger(dlog.VerbosePrintfLogger(
			log.Default(),
		)),
	)
	n := 10
	for i := 0; i < n; i++ {
		assert.Nil(
			t,
			dcr.AddJob(fmt.Sprintf("job_%d", i), "* * * * * *", &testGetJob{
				Name: fmt.Sprintf("job_%d", i),
			}),
		)
	}
	result := make(chan bool, 1)
	dcr.Start()
	go func() {
		// check function
		<-time.After(5 * time.Second)
		for i := 0; i < n; i++ {
			job, err := dcr.GetJob(fmt.Sprintf("job_%d", i), true)
			t.Log(job)
			s.Assert().Nil(err)
			s.Assert().NotNil(job.Job)
			job.Job.(*testGetJob).Reach()
		}

		for i := 0; i < n; i++ {
			job, err := dcr.GetJob(fmt.Sprintf("job_%d", i), true)
			s.Assert().Nil(err)
			s.Assert().Equal(1, job.Job.(*testGetJob).Reached)
		}
		result <- true
	}()
	s.Assert().True(<-result)
}

func (s *testDcronTestSuite) Test_AddJob_JobName_Duplicate() {
	t := s.T()
	rds := miniredis.RunT(t)
	defer rds.Close()
	redisCli := redis.NewClient(&redis.Options{
		Addr: rds.Addr(),
	})
	drv := redisdriver.NewDriver(redisCli)
	dcr := dcron.NewDcronWithOption(
		t.Name(),
		drv,
		dcron.CronOptionSeconds(),
		dcron.WithLogger(dlog.VerbosePrintfLogger(
			log.Default(),
		)),
	)
	s.Assert().Nil(dcr.AddJob("test1", "* * * * * *", &testGetJob{}))
	s.Assert().Equal(dcron.ErrJobExist, dcr.AddJob("test1", "* * * * * *", &testGetJob{}))
}

func TestDcronTestMain(t *testing.T) {
	suite.Run(t, new(testDcronTestSuite))
}
