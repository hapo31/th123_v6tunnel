package main

import (
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

	values := &Values{
		Mode:       "server",
		ClientPort: 10800,
		ServerPort: 10801,
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
	)

	var mw *MyMainWindow
	mwCfg := &MainWindow{
		Title:  "tunnel",
		Bounds: Rectangle{Width: 400, Height: 380},
		DataBinder: DataBinder{
			AssignTo:   &db,
			Name:       "values",
			DataSource: values,
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
						Title:     "モード",
						Layout:    HBox{},
						MaxSize:   Size{Height: 5},
						Alignment: AlignHCenterVNear,
						DataBinder: DataBinder{
							DataSource: values,
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
											submitButton.SetText("接続を待つ")
											serverGb.SetVisible(true)
											clientGb.SetVisible(false)
										},
									},
									{
										Name:  "modeClient",
										Text:  "クライアント",
										Value: "client",
										OnClicked: func() {
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
							Margins: Margins{10, 10, 10, 10},
						},
						Title: "サーバー設定",
						DataBinder: DataBinder{
							DataSource: values,
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
							Margins: Margins{10, 10, 10, 10},
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
								Text:       "接続先のIPアドレス",
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
							addr := fmt.Sprintf("[%s]:%d", ipAddr, values.ServerPort)
							walk.Clipboard().SetText(addr)
							statusBar.SetText("接続情報をクリップボードにコピーしました。")
						},
					},
					PushButton{
						AssignTo: &submitButton,
						Text:     "接続を待つ",
						OnClicked: func() {
							db.Submit()
							walk.MsgBox(mw, "Test", valueToStr(values), walk.MsgBoxIconInformation)
						},
					},
				},
			},
		},
	}
	mw, err := createMainWindow(mwCfg, &mw)

	if err != nil {
		log.Fatal(err)
	}

	mw.Run()
}

func valueToStr(v *Values) string {
	s := fmt.Sprintf("Mode: %s\n", v.Mode)
	s += fmt.Sprintf("ClientPort:%d\n", v.ClientPort)
	s += fmt.Sprintf("ServerPort:%d\n", v.ServerPort)
	s += fmt.Sprintf("ClientAddr:%s\n", v.ClientAddr)
	s += fmt.Sprintf("RemoteAddr:%s\n", v.RemoteAddr)

	return s
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

// func isGlobalIPAddr(ip* net.IP) {
// 	if ip[0]
// }
