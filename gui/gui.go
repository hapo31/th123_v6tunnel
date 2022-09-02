package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"strings"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
	"github.com/lxn/win"
)

type MyMainWindow struct {
	*walk.MainWindow
}

type Values struct {
	Mode       string
	ClientPort int
	ServerPort int
	RemoteAddr string
	ClientAddr string
}

func main() {

	err := runFakeServer()

	if err != nil {
		walk.MsgBox(nil, "Error", fmt.Sprintf(`
アプリ起動時にエラーが発生しました。
お手数ですが、この画面の内容を Win + Shift + Sキーを押してスクリーンショットを保存し、
開発者に問い合わせてください。
%s`, err.Error()), walk.MsgBoxIconError)
	}

	values := &Values{
		Mode:       "server",
		ClientPort: 10803,
		ServerPort: 10800,
		RemoteAddr: "",
		ClientAddr: "",
	}

	var (
		copyButton   *walk.PushButton
		submitButton *walk.PushButton
		serverGb     *walk.GroupBox
		clientGb     *walk.GroupBox
		clientAddrLe *walk.LineEdit
		serverAddrLe *walk.LineEdit
		statusBar    *walk.StatusBarItem
		db           *walk.DataBinder
		modeRadioGb  *walk.GroupBox
	)

	var (
		ctx    context.Context
		cancel context.CancelFunc
	)
	proxyStart := false

	var mw *MyMainWindow
	mwCfg := &MainWindow{
		Title:  "tunnel",
		Bounds: Rectangle{Width: 400, Height: 380},
		DataBinder: DataBinder{
			AssignTo:   &db,
			Name:       "values",
			DataSource: values,
			AutoSubmit: true,
		},
		Layout: VBox{
			Alignment: AlignHNearVNear,
		},
		StatusBarItems: []StatusBarItem{
			{
				AssignTo: &statusBar,
				Text:     "",
				Width:    400,
			},
		},
		Children: []Widget{
			VSplitter{
				Children: []Widget{
					GroupBox{
						AssignTo:  &modeRadioGb,
						Title:     "モード",
						Layout:    HBox{},
						MaxSize:   Size{Height: 5},
						Alignment: AlignHCenterVNear,
						DataBinder: DataBinder{
							DataSource: values,
							AutoSubmit: true,
						},
						Children: []Widget{
							RadioButtonGroup{
								DataMember: "Mode",
								Buttons: []RadioButton{
									{
										Name:  "modeServer",
										Text:  "サーバー",
										Value: "server",
										OnClicked: func() {
											values.Mode = "server"
											changeButtonTextByMode(submitButton, "server", false)
											serverGb.SetVisible(true)
											clientGb.SetVisible(false)
										},
									},
									{
										Name:  "modeClient",
										Text:  "クライアント",
										Value: "client",
										OnClicked: func() {
											values.Mode = "client"
											changeButtonTextByMode(submitButton, "client", false)
											submitButton.SetText("接続する")
											serverGb.SetVisible(false)
											clientGb.SetVisible(true)
										},
									},
								},
							},
						},
					},
					GroupBox{
						AssignTo: &serverGb,
						Layout: Grid{
							Columns: 3,
							Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10},
						},
						Title: "サーバー設定",
						DataBinder: DataBinder{
							DataSource: values,
							AutoSubmit: true,
						},
						Children: []Widget{
							Label{
								ColumnSpan: 1,
								Text:       "待ち受けポート",
							},
							NumberEdit{
								ColumnSpan: 2,
								Value:      Bind("ServerPort"),
								MaxValue:   65535,
								MinValue:   1,
							},
							Label{
								ColumnSpan: 1,
								Text:       "ゲーム側のポート",
							},
							NumberEdit{
								ColumnSpan: 2,
								Value:      Bind("ClientPort"),
								MaxValue:   65535,
								MinValue:   1,
							},
							Label{
								ColumnSpan: 3,
								Text:       "ゲーム側のアドレス(よくわからない場合は空欄)",
							},
							LineEdit{
								AssignTo:   &clientAddrLe,
								ColumnSpan: 3,
								Name:       "ClientAddr",
								Text:       Bind("ClientAddr"),
								OnEditingFinished: func() {
									values.ClientAddr = clientAddrLe.Text()
								},
							},
						},
					},
					GroupBox{
						AssignTo: &clientGb,
						Layout: Grid{
							Columns: 3,
							Margins: Margins{Left: 10, Top: 10, Right: 10, Bottom: 10},
						},
						Visible: values.Mode == "client",
						Title:   "クライアント設定",
						DataBinder: DataBinder{
							DataSource: values,
							AutoSubmit: true,
						},
						Children: []Widget{
							Label{
								ColumnSpan: 3,
								Text:       "接続先のアドレス",
							},
							LineEdit{
								AssignTo:   &serverAddrLe,
								ColumnSpan: 3,
								Text:       Bind("RemoteAddr"),
								Name:       "RemoteAddr",
								OnEditingFinished: func() {
									values.RemoteAddr = serverAddrLe.Text()
								},
							},
							Label{
								ColumnSpan: 1,
								Text:       "ゲーム側のポート",
							},
							NumberEdit{
								ColumnSpan: 2,
								Value:      Bind("ClientPort"),
								MaxValue:   65535,
								MinValue:   1,
							},
						},
					},
					PushButton{
						AssignTo: &copyButton,
						Text:     "接続情報をコピー",
						OnClicked: func() {
							db.Submit()
							addr := createClipboardText(values)
							walk.Clipboard().SetText(addr)
							statusBar.SetText("接続情報をクリップボードにコピーしました。")
						},
					},
					PushButton{
						AssignTo: &submitButton,
						Text:     "接続を待つ",
						OnClicked: func() {
							fmt.Printf("%+v\n", values)
							if !proxyStart {
								proxyStart = true
								ctx, cancel = context.WithCancel((context.Background()))

								go func() {
									modeRadioGb.SetEnabled(false)
									defer modeRadioGb.SetEnabled(true)
									errChan, err := startProxy(ctx, values)
									if err != nil {
										walk.MsgBox(*mw, "Error", fmt.Sprintf("エラーが発生しました。\n%s", err.Error()), walk.MsgBoxIconError)
										return
									}

									changeButtonTextByMode(submitButton, values.Mode, true)
									defer changeButtonTextByMode(submitButton, values.Mode, false)

									changeStatusText(statusBar, values.Mode, true)
									defer changeStatusText(statusBar, values.Mode, false)

									for {
										select {
										case <-ctx.Done():
											return
										case err := <-errChan:
											walk.MsgBox(*mw, "Error", fmt.Sprintf("エラーが発生しました。\n%s", err.Error()), walk.MsgBoxIconError)
											return
										}
									}

								}()
							} else {
								proxyStart = false
								cancel()
							}
						},
					},
				},
			},
		},
	}
	mw, err = createMainWindow(mwCfg, &mw)

	if err != nil {
		log.Fatal(err)
	}

	mw.Run()
}

func runFakeServer() error {
	var err error = nil
	// サーバーを立ててすぐに閉じることでファイアウォールの設定を出現させる。
	// 一度許可すれば以降は出ないはず
	port := 18888
	for port <= 25565 {
		addr, _ := net.ResolveUDPAddr("udp", fmt.Sprintf("0.0.0.0:%d", port))
		conn, err_ := net.ListenUDP("udp", addr)
		if err != nil {
			err = err_
			port += 1
			continue
		}
		defer conn.Close()
		break
	}

	return err
}

func startProxy(context context.Context, values *Values) (chan error, error) {

	var clientAddr string
	if values.ClientAddr == "" {
		clientAddr = fmt.Sprintf("127.0.0.1:%d", values.ClientPort)
	} else {
		clientAddr = values.ClientAddr
	}

	if values.Mode == "client" {
		return StartClient(context, clientAddr, values.RemoteAddr)
	} else {
		return StartServer(context, clientAddr, values.ServerPort)
	}
}

func createMainWindow(mwCfg *MainWindow, mw **MyMainWindow) (*MyMainWindow, error) {
	if *mw == nil {
		*mw = new(MyMainWindow)
	}
	mwCfg.AssignTo = &(*mw).MainWindow

	if err := mwCfg.Create(); err != nil {
		return nil, err
	}

	handle := (*mw).Handle()
	defaultStyle := win.GetWindowLong(handle, win.GWL_STYLE)
	win.SetWindowLong(handle, win.GWL_STYLE, defaultStyle&^win.WS_THICKFRAME)

	statusBarHandle := (*mw).StatusBar().Handle()
	statusBarDefaultStyle := win.GetWindowLong(statusBarHandle, win.GWL_STYLE)
	win.SetWindowLong(statusBarHandle, win.GWL_STYLE, statusBarDefaultStyle&^win.SBARS_SIZEGRIP)

	return *mw, nil
}

func changeButtonTextByMode(btn *walk.PushButton, mode string, startProxy bool) {
	if mode == "server" {
		if startProxy {
			btn.SetText("接続待ちを解除")
		} else {
			btn.SetText("接続を待つ")
		}
	} else {
		if startProxy {
			btn.SetText("接続待ちを解除")
		} else {
			btn.SetText("接続する")
		}
	}
}

func changeStatusText(statusBar *walk.StatusBarItem, mode string, startProxy bool) {
	if mode == "server" {
		if startProxy {
			statusBar.SetText("接続待ち中...")
		} else {
			statusBar.SetText("")
		}
	} else {
		if startProxy {
			statusBar.SetText("クライアントに接続中...")
		} else {
			statusBar.SetText("")
		}
	}
}

func createClipboardText(values *Values) (addr string) {
	if values.Mode == "server" {
		var ipAddr string
		first := false
		addrs, _ := net.InterfaceAddrs()

		for _, add := range addrs {
			networkIp, ok := add.(*net.IPNet)
			if ok && !networkIp.IP.IsLoopback() && networkIp.IP.To4() == nil && networkIp.IP.To16() != nil {
				if first {
					fmt.Println(add)
					ipAddr = strings.Split(add.String(), "/")[0]
					break
				}
				first = true
			}
		}
		addr = fmt.Sprintf("[%s]:%d", ipAddr, values.ServerPort)
	} else {
		addr = fmt.Sprintf("127.0.0.1:%d", values.ClientPort)
	}
	return addr
}
