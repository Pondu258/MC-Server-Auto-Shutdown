// main.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

const configDir  = "MSAS"
const configFile = "MSAS/config.json"
const logFile    = "MSAS/System.log"

type Config struct {
	ServerFolder     string `json:"server_folder"`
	ServerJar        string `json:"server_jar"`
	CountdownSeconds int    `json:"countdown_seconds"`
	ShutdownTimeStart string `json:"shutdown_time_start"` // "HH:MM"
	ShutdownTimeEnd   string `json:"shutdown_time_end"`   // "HH:MM"
}

var config Config

// ── 設定 ────────────────────────────────────────────

func loadConfig() {
	config = Config{
		ServerFolder:      "server",
		ServerJar:         "forge-server.jar",
		CountdownSeconds:  60,
		ShutdownTimeStart: "02:00",
		ShutdownTimeEnd:   "08:00",
	}
	data, err := os.ReadFile(configFile)
	if err != nil {
		return
	}
	json.Unmarshal(data, &config)
}

func saveConfig() {
	os.MkdirAll(configDir, 0755)
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configFile, data, 0644)
}

func parseHHMM(s string) (int, int, error) {
	parts := strings.Split(s, ":")
	if len(parts) != 2 {
		return 0, 0, fmt.Errorf("invalid format")
	}
	h, err1 := strconv.Atoi(parts[0])
	m, err2 := strconv.Atoi(parts[1])
	if err1 != nil || err2 != nil || h < 0 || h > 23 || m < 0 || m > 59 {
		return 0, 0, fmt.Errorf("invalid value")
	}
	return h, m, nil
}

func isInShutdownWindow() bool {
	startH, startM, err1 := parseHHMM(config.ShutdownTimeStart)
	endH, endM, err2 := parseHHMM(config.ShutdownTimeEnd)
	if err1 != nil || err2 != nil {
		return true // パース失敗時は常にシャットダウン許可
	}

	now := time.Now()
	nowMinutes := now.Hour()*60 + now.Minute()
	startMinutes := startH*60 + startM
	endMinutes := endH*60 + endM

	if startMinutes <= endMinutes {
		// 例: 02:00 ～ 08:00
		return nowMinutes >= startMinutes && nowMinutes < endMinutes
	}
	// 日をまたぐ場合: 例: 22:00 ～ 06:00
	return nowMinutes >= startMinutes || nowMinutes < endMinutes
}

// ── セットアップ ─────────────────────────────────────

func readLine(reader *bufio.Reader, prompt string, fallback string) string {
	fmt.Printf("%s (%s): ", prompt, fallback)
	input, _ := reader.ReadString('\n')
	input = strings.TrimSpace(input)
	if input == "" {
		return fallback
	}
	return input
}

func setup() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=============================================")
	fmt.Println("  MC Server Auto Shutdown System")
	fmt.Println("=============================================")
	fmt.Printf("\n現在の設定:\n")
	fmt.Printf("  サーバーフォルダ  : %s\n", config.ServerFolder)
	fmt.Printf("  JARファイル名    : %s\n", config.ServerJar)
	fmt.Printf("  シャットダウンまで: %d秒\n", config.CountdownSeconds)
	fmt.Printf("  シャットダウン時間帯: %s ～ %s\n", config.ShutdownTimeStart, config.ShutdownTimeEnd)
	fmt.Print("\n変更しますか？ (y/Enter): ")
	input, _ := reader.ReadString('\n')

	if strings.TrimSpace(input) != "y" {
		fmt.Printf("\n設定をそのまま使用します。Enterを押すとサーバーを起動します...")
		reader.ReadString('\n')
		return
	}

	// サーバーフォルダ
	config.ServerFolder = readLine(reader, "サーバーフォルダ名", config.ServerFolder)

	// JARファイル名
	config.ServerJar = readLine(reader, "サーバーのJARファイル名", config.ServerJar)

	// 待機時間
	for {
		secStr := readLine(reader, "シャットダウンまでの待機時間（秒）", strconv.Itoa(config.CountdownSeconds))
		sec, err := strconv.Atoi(secStr)
		if err == nil && sec > 0 {
			config.CountdownSeconds = sec
			break
		}
		fmt.Println("  ※ 正しい数値を入力してください")
	}

	// シャットダウン時間帯
	for {
		s := readLine(reader, "シャットダウン開始時刻 (HH:MM)", config.ShutdownTimeStart)
		_, _, err := parseHHMM(s)
		if err == nil {
			config.ShutdownTimeStart = s
			break
		}
		fmt.Println("  ※ HH:MM 形式で入力してください (例: 02:00)")
	}
	for {
		s := readLine(reader, "シャットダウン終了時刻 (HH:MM)", config.ShutdownTimeEnd)
		_, _, err := parseHHMM(s)
		if err == nil {
			config.ShutdownTimeEnd = s
			break
		}
		fmt.Println("  ※ HH:MM 形式で入力してください (例: 08:00)")
	}
    
	os.MkdirAll(configDir, 0755)
	saveConfig()
	fmt.Println("\n設定を保存しました！")
	fmt.Printf("\n設定完了:\n  フォルダ: %s / JAR: %s\n  停止後 %d秒 でシャットダウン (%s ～ %s の間のみ)\n",
		config.ServerFolder, config.ServerJar, config.CountdownSeconds,
		config.ShutdownTimeStart, config.ShutdownTimeEnd)
	fmt.Print("\nEnterを押すとサーバーを起動します...")
	reader.ReadString('\n')
}

// ── サーバー起動 ─────────────────────────────────────

func startServer() (normalExit bool) {
	jarPath := filepath.Join(config.ServerFolder, config.ServerJar)
	fmt.Printf("\nサーバーを起動します... (%s)\n\n", jarPath)

	cmd := exec.Command("java", "-Xmx4G", "-jar", jarPath, "nogui")
	cmd.Dir = config.ServerFolder
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	err := cmd.Run()
	if err != nil {
		fmt.Println("サーバーが異常終了しました:", err)
		return false
	}
	fmt.Println("サーバーが正常に終了しました")
	return true
}

// ── カウントダウン ───────────────────────────────────

func countdown() bool {
	fmt.Println("\n=============================================")
	fmt.Println("  MCサーバーが停止しました")
	fmt.Printf("  %d秒後にPCをシャットダウンします\n", config.CountdownSeconds)
	fmt.Println("  キャンセルするには Enter を押してください")
	fmt.Println("=============================================\n")

	cancelled := make(chan struct{})
	go func() {
		bufio.NewReader(os.Stdin).ReadString('\n')
		close(cancelled)
	}()

	for i := config.CountdownSeconds; i > 0; i-- {
		select {
		case <-cancelled:
			fmt.Println("\nシャットダウンをキャンセルしました")
			return false
		default:
			fmt.Printf("\r残り %3d秒...", i)
			time.Sleep(time.Second)
		}
	}

	select {
	case <-cancelled:
		fmt.Println("\nシャットダウンをキャンセルしました")
		return false
	default:
		return true
	}
}

// ── シャットダウン ───────────────────────────────────

func shutdown() {
	fmt.Println("\nシャットダウンします...")
	cmd := exec.Command(`C:\Windows\System32\shutdown.exe`, "/s", "/t", "0")
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("エラー:", err)
		fmt.Print("Enterを押すと終了します...")
		bufio.NewReader(os.Stdin).ReadString('\n')
	}
}

// ── ログ記録 ─────────────────────────────────────────

func writeLog(normalExit bool, stopTime time.Time) {
	const maxEntries = 5
	
	os.MkdirAll(configDir, 0755) 
	
	// 既存のログを読み込む
	var lines []string
	data, err := os.ReadFile(logFile)
	if err == nil {
		for _, line := range strings.Split(string(data), "\n") {
			if strings.TrimSpace(line) != "" {
				lines = append(lines, line)
			}
		}
	}

	// 新しいエントリを追加
	exitStatus := "異常終了"
	if normalExit {
		exitStatus = "正常終了"
	}
	newEntry := fmt.Sprintf("[%s] サーバー停止 | 状態: %s | 停止時刻: %s",
		stopTime.Format("2006-01-02 15:04:05"),
		exitStatus,
		stopTime.Format("15:04:05"),
	)
	lines = append(lines, newEntry)

	// 直近5件だけ残す
	if len(lines) > maxEntries {
		lines = lines[len(lines)-maxEntries:]
	}

	// 書き直す
	f, err := os.Create(logFile)
	if err != nil {
		fmt.Println("ログの書き込みに失敗しました:", err)
		return
	}
	defer f.Close()
	f.WriteString(strings.Join(lines, "\n") + "\n")

	fmt.Printf("\nログを記録しました: %s\n", logFile)
}

// ── メイン ───────────────────────────────────────────

func main() {
	loadConfig()
	setup()

	normalExit := startServer()
	stopTime := time.Now()

	writeLog(normalExit, stopTime)

	if !isInShutdownWindow() {
		fmt.Printf("\n現在時刻 %s はシャットダウン時間帯 (%s ～ %s) 外のため、シャットダウンしません\n",
			stopTime.Format("15:04"),
			config.ShutdownTimeStart,
			config.ShutdownTimeEnd,
		)
		fmt.Print("Enterを押すと終了します...")
		bufio.NewReader(os.Stdin).ReadString('\n')
		return
	}

	if countdown() {
		shutdown()
	}
}