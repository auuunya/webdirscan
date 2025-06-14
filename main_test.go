package main

import (
	"os"
	"strings"
	"testing"
	"time"
)

func TestScan_Run(t *testing.T) {
	// 临时字典文件
	dictFile := "test_dict.txt"
	outputFile := "test_output.txt"

	// 准备一个假的字典内容
	dictContent := []string{
		"/robots.txt", // 有效地址
		"/index.html", // 无效地址
	}
	// 写入字典文件
	f, _ := os.Create(dictFile)
	defer os.Remove(dictFile) // 用完删除
	for _, line := range dictContent {
		f.WriteString(line + "\n")
	}
	f.Close()

	// 初始化扫描器，URL 用 httpbin.org，线程设为2
	scan, err := NewScan("http://httpbin.org", 2, outputFile)
	if err != nil {
		t.Fatalf("初始化失败: %v", err)
	}

	// 加载字典并执行
	go scan.LoadDict(dictFile)
	scan.Run()
	scan.Close()
	// 检查输出文件
	time.Sleep(2 * time.Second) // 等待写入完成
	content, err := os.ReadFile(outputFile)
	defer os.Remove(outputFile) // 清理
	if err != nil {
		t.Fatalf("读取输出失败: %v", err)
	}
	output := string(content)

	// 断言内容包含 ✅ 和 ❌
	if !strings.Contains(output, "✅") && !strings.Contains(output, "❌") {
		t.Errorf("输出内容异常: %s", output)
	}
}

func BenchmarkScan_Run(b *testing.B) {
	dictFile := "bench_dict.txt"
	outputFile := "bench_output.txt"

	// 生成一个中等大小的字典（100条）
	f, _ := os.Create(dictFile)
	defer os.Remove(dictFile) // 清理
	for i := 0; i < 100; i++ {
		f.WriteString("/status/200\n")
	}
	f.Close()

	b.ResetTimer() // 重置定时器，忽略前面的准备时间

	for i := 0; i < b.N; i++ {
		scan, err := NewScan("http://httpbin.org", 10, outputFile)
		if err != nil {
			b.Fatalf("初始化失败: %v", err)
		}

		go scan.LoadDict(dictFile)
		scan.Run()

		time.Sleep(time.Millisecond * 100) // 避免 IO 冲突
		os.Remove(outputFile)              // 每轮删除
	}
}
