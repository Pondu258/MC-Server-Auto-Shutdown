// main.go
package main

import (
	"bufio"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"strconv"
	"strings"
	"time"
)

const configFile = "config.json"

type Config struct {
	ServerJar        string `json:"server_jar"`
	CountdownSeconds int    `json:"countdown_seconds"`
}

var config Config

func loadConfig() {
	data, err := os.ReadFile(configFile)
	if err != nil {
		// ファイルがなければデフォルト値をセット
		config = Config{
			ServerJar:        "forge-server.jar",
			CountdownSeconds: 60,
		}
		return
	}
	json.Unmarshal(data, &config)
}

func saveConfig() {
	data, _ := json.MarshalIndent(config, "", "  ")
	os.WriteFile(configFile, data, 0644)
}

func setup() {
	reader := bufio.NewReader(os.Stdin)

	fmt.Println("=============================================")
	fmt.Println("  MC Server Auto Shutdown Launcher")
	fmt.Println("=============================================")
	fmt.Printf("\n現在の設定:\n")
	fmt.Printf("  JARファイル名    : %s\n", config.ServerJar)
	fmt.Printf("  シャットダウンまで: %d秒\n", config.CountdownSeconds)
	fmt.Println("\n変更しますか？ (y/Enter): ")
	input, _ := reader.ReadString('\n')

	if strings.TrimSpace(input) != "y" {
		return
	}

	// JARファイル名
	fmt.Printf("サーバーのJARファイル名 (%s): ", config.ServerJar)
	jar, _ := reader.ReadString('\n')
	jar = strings.TrimSpace(jar)
	if jar != "" {
		config.ServerJar = jar
	}

	// 待機時間
	for {
		fmt.Printf("シャットダウンまでの待機時間（秒）(%d): ", config.CountdownSeconds)
		secInput, _ := reader.ReadString('\n')
		secInput = strings.TrimSpace(secInput)
		if secInput == "" {
			break // 変更なし
		}
		sec, err := strconv.Atoi(secInput)
		if err == nil && sec > 0 {
			config.CountdownSeconds = sec
			break
		}
		fmt.Println("  ※ 正しい数値を入力してください")
	}

	saveConfig()
	fmt.Println("\n設定を保存しました！")

	fmt.Printf("\n設定完了: %s を起動、サーバー停止後 %d秒 でシャットダウン\n", config.ServerJar, config.CountdownSeconds)
	fmt.Print("Enterを押すとサーバーを起動します...")
	reader.ReadString('\n')
}

func startServer() {
	fmt.Println("\nサーバーを起動します...\n")
	cmd := exec.Command("java", "-Xmx4G", "-jar", config.ServerJar, "nogui")
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Run()
	if err != nil {
		fmt.Println("サーバーが終了しました:", err)
	} else {
		fmt.Println("サーバーが正常に終了しました")
	}
}

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

func shutdown() {
	fmt.Println("\nシャットダウンします...")
	cmd := exec.Command("shutdown", "/s", "/t", "0")
	cmd.Run()
}

func main() {
	loadConfig()
	setup()
	startServer()
	if countdown() {
		shutdown()
	}
}
