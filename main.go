package main

import (
	"bufio"
	"flag"
	"fmt"
	"net/http"
	"os"
	"strings"
	"sync"
	"time"
)

type Scan struct {
	Url       string
	Stop      bool
	Thread    int
	wg        *sync.WaitGroup
	channel   chan string
	ScanCount int
	ErrCount  int
	output    *os.File
	client    *http.Client
	lock      *sync.Mutex
}

func NewScan(scanUrl string, thread int, output string) (*Scan, error) {
	file, err := os.OpenFile(output, os.O_CREATE|os.O_APPEND|os.O_WRONLY, 0644)
	if err != nil {
		return nil, err
	}
	file.Truncate(0)
	scan := &Scan{
		Url:     scanUrl,
		Stop:    false,
		Thread:  thread,
		channel: make(chan string),
		wg:      &sync.WaitGroup{},
		output:  file,
		lock:    &sync.Mutex{},
	}
	scan.client = &http.Client{
		Timeout: 5 * time.Second,
	}
	return scan, nil
}

func (s *Scan) LoadDict(dict_file string) {
	f, err := os.Open(dict_file)
	if err != nil {
		return
	}
	defer f.Close()
	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := scanner.Text()
		if strings.TrimSpace(line) == "" || strings.HasPrefix(line, "#") {
			continue
		}
		s.channel <- strings.TrimSpace(line)
	}
	close(s.channel)
}

func (s *Scan) Close() {
	s.output.Close()
}

func (s *Scan) WriteFile(line string) {
	s.lock.Lock()
	defer s.lock.Unlock()
	s.output.WriteString(line + "\n")
}

func (s *Scan) Run() {
	for i := 0; i < s.Thread; i++ {
		s.wg.Add(1)
		go func(id int) {
			defer s.wg.Done()
			for ch := range s.channel {
				if s.Stop {
					return
				}
				rawUrl := fmt.Sprintf("%s%s", s.Url, ch)
				err := s.request(rawUrl)
				if err != nil {
					s.WriteFile(fmt.Sprintf("❌ [%s]不存在.", rawUrl))
				} else {
					s.WriteFile(fmt.Sprintf("✅ [%s]存在.", rawUrl))
				}
			}
		}(i)
	}
	s.wg.Wait()
}

func (s *Scan) request(rawURL string) error {
	u, err := http.NewRequest("GET", rawURL, nil)
	if err != nil {
		return err
	}
	u.Header.Add("User-Agent", "Mozilla/4.0 (compatible; MSIE 7.0; Windows NT 6.1;")
	u.Header.Add("Cache-Control", "no-cache")
	u.Header.Add("Referer", "www.baidu.com")
	response, err := s.client.Do(u)
	if err != nil {
		return err
	}
	defer response.Body.Close()
	if response.StatusCode == http.StatusOK {
		return nil
	}
	return fmt.Errorf("请求异常: %v", response.StatusCode)
}

func initParse() (string, string, string, int) {
	var url, dict, output string
	flag.StringVar(&url, "u", "", "The website to be scanned")
	flag.StringVar(&dict, "d", "dict/dict.txt", "Dictionary for scanning")
	flag.StringVar(&output, "o", "scanned.txt", "Results saved files")
	var thread int
	flag.IntVar(&thread, "t", 8, "Number of threads running the program")
	flag.Parse()
	return url, dict, output, thread
}
func main() {
	url, dict, output, thread := initParse()

	scan, err := NewScan(url, thread, output)
	if err != nil {
		fmt.Printf("创建扫描任务失败: %e\n", err)
	}
	go scan.LoadDict(dict) // 开 goroutine 防止阻塞（也可以改用缓冲通道）
	scan.Run()
	scan.Close()
	fmt.Printf("✅ 执行完毕.")
}
