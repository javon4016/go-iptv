package until

import (
	"context"
	"go-iptv/dao"
	"go-iptv/models"
	"log"
	"strconv"
	"sync"
	"time"
)

var Cache *SignalExecutor

type SignalExecutor struct {
	delay     time.Duration
	execFunc  func(ctx context.Context)
	signalCh  chan struct{}
	stopCh    chan struct{}
	cancel    context.CancelFunc
	timerMu   sync.Mutex
	waitTimer *time.Timer
}

// 创建 SignalExecutor 实例
func NewSignalExecutor(delay time.Duration, execFunc func(ctx context.Context)) *SignalExecutor {
	return &SignalExecutor{
		delay:    delay,
		execFunc: execFunc,
		signalCh: make(chan struct{}, 1),
		stopCh:   make(chan struct{}),
	}
}

// 启动信号监听器
func (s *SignalExecutor) Start() {
	go func() {
		for {
			select {
			case <-s.stopCh:
				log.Println("🛑 EPG缓存重建定时任务 已停止")
				return
			case <-s.signalCh:
				s.handleSignal()
			}
		}
	}()
}

// 外部调用此函数发出信号
func (s *SignalExecutor) Rebuild() {
	select {
	case s.signalCh <- struct{}{}:
	default:
		// 若通道已满，忽略（表示已有信号等待处理）
	}
}

// 停止执行器
func (s *SignalExecutor) Stop() {
	close(s.stopCh)
	s.timerMu.Lock()
	if s.waitTimer != nil {
		s.waitTimer.Stop()
	}
	if s.cancel != nil {
		s.cancel()
	}
	s.timerMu.Unlock()
}

// 内部信号处理逻辑
func (s *SignalExecutor) handleSignal() {
	s.timerMu.Lock()
	defer s.timerMu.Unlock()

	// 如果任务正在执行 → 先中断
	if s.cancel != nil {
		log.Println("⛔ 中断当前执行EPG缓存重建任务")
		s.cancel()
		s.cancel = nil
	}

	// 若已有计时器 → 重置计时
	if s.waitTimer != nil {
		s.waitTimer.Stop()
		s.waitTimer.Reset(s.delay)
		log.Println("🔁 重置EPG缓存重建信号等待 10 秒")
		return
	}

	// 新建计时器
	log.Println("⏳ 收到EPG缓存重建信号，10 秒后执行")
	s.waitTimer = time.AfterFunc(s.delay, func() {
		s.timerMu.Lock()
		ctx, cancel := context.WithCancel(context.Background())
		s.cancel = cancel
		s.waitTimer = nil
		s.timerMu.Unlock()

		log.Println("🚀 开始执行EPG缓存重建任务")
		s.execFunc(ctx)
	})
}

func doRebuild(ctx context.Context) {
	select {
	case <-ctx.Done():
		log.Println("⚠️ 重建任务被中断")
		return
	default:
		makeMealsXmlCacheAll()
		log.Println("✅ EPG缓存重建任务执行完成")
		cfg := dao.GetConfig()
		if cfg.Resolution.Auto == 1 && dao.Lic.Tpye != 0 {
			log.Println("🚀 开始执行分辨率识别任务")
			log.Println("开始执行分辨率测试，测试期间cpu、内存占用会较高，请耐心等待，中断执行请关闭自动测试并重启引擎")
			res, err := dao.WS.SendWS(dao.Request{Action: "testResolutionAll"}) //测试分辨率
			if err != nil {
				log.Println("分辨率测试失败:", err)
			} else if res.Code != 1 {
				log.Println("分辨率测试失败:", res.Msg)
			} else {
				log.Println("分辨率测试任务执行中...")

				res, _ := dao.WS.SendWS(dao.Request{Action: "getTestStatus"}) //获取测试状态
				for res.Code != 1 {
					time.Sleep(5 * time.Second)
					res, _ = dao.WS.SendWS(dao.Request{Action: "getTestStatus"}) //获取测试状态
				}
				log.Println("分辨率测试完成")
				log.Println("🚀 重新执行EPG缓存重建")
				dao.Cache.Clear() //清除缓存
				makeMealsXmlCacheAll()
				log.Println("✅ EPG缓存重建任务执行完成")
			}
		}
	}
}

func InitCacheRebuild() {
	// 创建执行器：任务为打印模拟执行
	Cache = NewSignalExecutor(10*time.Second, doRebuild)
	log.Println("🔧 EPG缓存重建任务初始化完成")

	// 启动执行器
	Cache.Start()

	select {}
}

func CleanMealsXmlCacheAll() {
	var meals []models.IptvMeals
	dao.DB.Model(&models.IptvMeals{}).Find(&meals)
	for _, meal := range meals {
		dao.Cache.Delete("rssEpgXml_" + strconv.FormatInt(meal.ID, 10))
	}
	Cache.Rebuild()
}

func makeMealsXmlCacheAll() {
	var meals []models.IptvMeals
	dao.DB.Model(&models.IptvMeals{}).Where("status = 1").Find(&meals)
	for _, meal := range meals {
		GetEpg(meal.ID)
	}
}

func CleanMealsXmlCacheOne(id int64) {
	log.Println("删除套餐EPG订阅缓存: ", id)
	dao.Cache.Delete("rssEpgXml_" + strconv.FormatInt(id, 10))
	GetEpg(id)
}

func CleanMealsTxtCacheAll() {
	var meals []models.IptvMeals
	dao.DB.Model(&models.IptvMeals{}).Find(&meals)
	for _, meal := range meals {
		dao.Cache.Delete("rssMealTxt_" + strconv.FormatInt(meal.ID, 10))
	}

	CleanMealsXmlCacheAll()
}

func CleanMealsTxtCacheOne(id int64) {
	log.Println("删除套餐TXT订阅缓存: ", id)
	dao.Cache.Delete("rssMealTxt_" + strconv.FormatInt(id, 10))
	CleanMealsXmlCacheOne(id)
}

func CleanAutoCacheAll() {
	var ca []models.IptvCategory
	dao.DB.Model(&models.IptvCategory{}).Where("enable = 1 and type = ?", "auto").Find(&ca)
	for _, ca := range ca {
		log.Println("删除自动聚合缓存: ", ca.Name)
		dao.Cache.Delete("autoCategory_" + strconv.FormatInt(ca.ID, 10))
	}
	CleanMealsTxtCacheAll()
}
