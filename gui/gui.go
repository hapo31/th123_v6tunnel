package main

import (
	"fmt"
	"log"

	"github.com/lxn/walk"
	. "github.com/lxn/walk/declarative"
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

func (v *Values) GetSubmitButtonText() string {
	if v.Mode == "server" {
		return "接続待機"
	} else {
		return "接続"
	}
}

func main() {

	values := &Values{
		Mode:       "server",
		ClientPort: 10800,
		ServerPort: 10801,
	}

	var submitButton *walk.PushButton
	var serverGb *walk.GroupBox
	var clientGb *walk.GroupBox

	mw := new(MyMainWindow)
	if err := (MainWindow{
		AssignTo: &mw.MainWindow,
		Title:    "tunnel",
		Bounds:   Rectangle{Width: 400, Height: 600},
		DataBinder: DataBinder{
			DataSource: values,
			AutoSubmit: true,
		},
		Layout: VBox{
			Alignment: AlignHNearVNear,
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
								ColumnSpan: 3,
								Name:       "ClientAddr",
								Text:       Bind("ClientAddr"),
								OnTextChanged: func() {

								},
							},
						},
					},
					GroupBox{
						AssignTo: &clientGb,
						Layout: Grid{
							Columns: 3,
						},
						Visible: values.Mode == "client",
						Title:   "クライアント設定",
						Children: []Widget{
							Label{
								ColumnSpan: 3,
								Text:       "接続先のIPアドレス",
							},
							LineEdit{
								ColumnSpan: 3,
								Text:       Bind("RemoteAddr"),
								Name:       "RemoteAddr",
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
						AssignTo: &submitButton,
						Text:     "接続を待つ",
						OnClicked: func() {
							walk.MsgBox(mw, "Test", valueToStr(values), walk.MsgBoxIconInformation)
						},
					},
				},
			},
		},
	}).Create(); err != nil {
		log.Fatal(err)
	}

	mw.Run()
}

func valueToStr(v *Values) string {
	// s := fmt.Sprintf("Mode: %s\n", v.Mode)
	// s += fmt.Sprintf("ClientPort:%d\n", v.ClientPort)
	// s += fmt.Sprintf("ServerPort:%d\n", v.ServerPort)
	// s += fmt.Sprintf("ClientAddr:%s\n", v.ClientAddr)
	// s += fmt.Sprintf("RemoteAddr:%s\n", v.RemoteAddr)

	s := fmt.Sprintf("%+v", v)

	return s
}
