package main

import (
	"context"
	"github.com/andlabs/ui"
	_ "github.com/andlabs/ui/winmanifest"
	"github.com/go-playground/validator/v10"
	"github.com/panjf2000/gnet/pool/goroutine"
	log "github.com/sirupsen/logrus"
	"github.com/spf13/viper"
	"go.uber.org/fx"
	"os"
	"path/filepath"
	"relaybaton/pkg/config"
	"relaybaton/pkg/core"
)

var mainWin *ui.Window

var spinBoxClientPort *ui.Spinbox
var entryClientServer *ui.Entry
var entryClientUsername *ui.Entry
var passwordEntryClientPassword *ui.Entry
var radioButtonsClientRoute *ui.RadioButtons
var comboBoxDNSType *ui.Combobox
var entryDNSServer *ui.Entry
var entryDNSAddr *ui.Entry
var entryLogFile *ui.Entry
var comboBoxLogLevel *ui.Combobox

var buttonRun *ui.Button
var buttonStop *ui.Button
var buttonOpen *ui.Button
var buttonSave *ui.Button

var conf *config.ConfigGo
var app *fx.App
var ctx context.Context
var cancel context.CancelFunc

func setupUI() {
	mainWin = ui.NewWindow("Relaybaton", 1, 1, true)
	mainWin.SetMargined(true)
	mainWin.OnClosing(func(*ui.Window) bool {
		mainWin.Destroy()
		ui.Quit()
		return false
	})
	ui.OnShouldQuit(func() bool {
		mainWin.Destroy()
		return true
	})

	hbox := ui.NewVerticalBox()
	hbox.SetPadded(true)
	mainWin.SetChild(hbox)
	grid := ui.NewGrid()
	grid.SetPadded(true)
	hbox.Append(grid, true)

	groupClient := ui.NewGroup("Client")
	groupClient.SetMargined(true)
	gridClient := ui.NewGrid()
	gridClient.SetPadded(true)
	groupClient.SetChild(gridClient)
	labelClientPort := ui.NewLabel("Port")
	spinBoxClientPort = ui.NewSpinbox(0, 65535)
	gridClient.Append(labelClientPort, 0, 0, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridClient.Append(spinBoxClientPort, 1, 0, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	labelClientServer := ui.NewLabel("Server")
	entryClientServer = ui.NewEntry()
	gridClient.Append(labelClientServer, 0, 1, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridClient.Append(entryClientServer, 1, 1, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	labelClientUsername := ui.NewLabel("Username")
	entryClientUsername = ui.NewEntry()
	gridClient.Append(labelClientUsername, 0, 2, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridClient.Append(entryClientUsername, 1, 2, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	labelClientPassword := ui.NewLabel("Password")
	passwordEntryClientPassword = ui.NewPasswordEntry()
	gridClient.Append(labelClientPassword, 0, 3, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridClient.Append(passwordEntryClientPassword, 1, 3, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	labelClientRoute := ui.NewLabel("Route")
	radioButtonsClientRoute = ui.NewRadioButtons()
	radioButtonsClientRoute.Append("China only")
	radioButtonsClientRoute.Append("Proxy all")
	gridClient.Append(labelClientRoute, 0, 4, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridClient.Append(radioButtonsClientRoute, 1, 4, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	grid.Append(groupClient, 0, 1, 2, 1, true, ui.AlignCenter, true, ui.AlignCenter)

	groupDNS := ui.NewGroup("DNS")
	groupDNS.SetMargined(true)
	gridDNS := ui.NewGrid()
	gridDNS.SetPadded(true)
	groupDNS.SetChild(gridDNS)
	labelDNSType := ui.NewLabel("Type")
	comboBoxDNSType = ui.NewCombobox()
	comboBoxDNSType.Append("plaintext")
	comboBoxDNSType.Append("DNS over TLS")
	comboBoxDNSType.Append("DNS over HTTPS")
	gridDNS.Append(labelDNSType, 0, 0, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridDNS.Append(comboBoxDNSType, 1, 0, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	labelDNSServer := ui.NewLabel("Server")
	entryDNSServer = ui.NewEntry()
	gridDNS.Append(labelDNSServer, 0, 1, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridDNS.Append(entryDNSServer, 1, 1, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	labelDNSAddr := ui.NewLabel("Address")
	entryDNSAddr = ui.NewEntry()
	gridDNS.Append(labelDNSAddr, 0, 2, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridDNS.Append(entryDNSAddr, 1, 2, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	grid.Append(groupDNS, 0, 2, 2, 1, true, ui.AlignCenter, true, ui.AlignCenter)

	groupLog := ui.NewGroup("Log")
	groupLog.SetMargined(true)
	gridLog := ui.NewGrid()
	gridLog.SetPadded(true)
	groupLog.SetChild(gridLog)
	labelLogFile := ui.NewLabel("File")
	entryLogFile = ui.NewEntry()
	entryLogFile.SetReadOnly(true)
	buttonLogFile := ui.NewButton("...")
	buttonLogFile.OnClicked(func(button *ui.Button) {
		filename := ui.OpenFile(mainWin)
		if filename != "" {
			entryLogFile.SetText(filename)
		}
	})
	gridLog.Append(labelLogFile, 0, 0, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridLog.Append(entryLogFile, 1, 0, 1, 1, true, ui.AlignFill, true, ui.AlignCenter)
	gridLog.Append(buttonLogFile, 2, 0, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	labelLogLevel := ui.NewLabel("Level")
	comboBoxLogLevel = ui.NewCombobox()
	comboBoxLogLevel.Append("Panic")
	comboBoxLogLevel.Append("Fatal")
	comboBoxLogLevel.Append("Error")
	comboBoxLogLevel.Append("Warn")
	comboBoxLogLevel.Append("Info")
	comboBoxLogLevel.Append("Debug")
	comboBoxLogLevel.Append("Trace")
	gridLog.Append(labelLogLevel, 0, 1, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	gridLog.Append(comboBoxLogLevel, 1, 1, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	grid.Append(groupLog, 0, 3, 2, 1, true, ui.AlignCenter, true, ui.AlignCenter)

	buttonOpen = ui.NewButton("Open")
	buttonOpen.OnClicked(func(button *ui.Button) {
		filename := ui.OpenFile(mainWin)
		if filename != "" {
			viper.SetConfigName(filepath.Base(filename))
			viper.SetConfigType(filepath.Ext(filename)[1:])
			viper.AddConfigPath(filepath.Dir(filename) + "/")
			var err error
			conf, err = config.NewConfClient()
			if err != nil {
				//TODO
				log.Error(err)
				return
			}
			ApplyConf(conf)
		}
	})
	buttonSave = ui.NewButton("Save")
	buttonSave.OnClicked(func(button *ui.Button) {
		var err error
		conf, err = GetConf()
		if err != nil {
			log.Error(err)
			ui.MsgBoxError(mainWin, "Error", err.Error())
			return
		}
		filename := ui.SaveFile(mainWin)
		if filename != "" {
			err = conf.Save(filename)
			if err != nil {
				log.Error(err)
				ui.MsgBoxError(mainWin, "Error", err.Error())
				return
			}
			ApplyConf(conf)
		}
	})
	buttonRun = ui.NewButton("Run")
	buttonStop = ui.NewButton("Stop")
	buttonStop.Disable()
	buttonRun.OnClicked(func(button *ui.Button) {
		spinBoxClientPort.Disable()
		entryClientServer.Disable()
		entryClientUsername.Disable()
		passwordEntryClientPassword.Disable()
		radioButtonsClientRoute.Disable()
		comboBoxDNSType.Disable()
		entryDNSServer.Disable()
		entryDNSAddr.Disable()
		entryLogFile.Disable()
		comboBoxLogLevel.Disable()
		buttonOpen.Disable()
		buttonSave.Disable()
		buttonRun.Disable()
		buttonStop.Enable()
		err := conf.Save(viper.ConfigFileUsed())
		if err != nil {
			log.Error(err)
			ui.MsgBoxError(mainWin, "Error", err.Error())
			return
		}

		ctx, cancel = context.WithCancel(context.Background())
		defer cancel()
		var client *core.Client
		app = fx.New(
			fx.Provide(
				core.NewClient,
				config.NewConfClient,
				goroutine.Default,
			),
			fx.Logger(log.StandardLogger()),
			fx.Invoke(config.InitLog, config.InitDNS),
			fx.Populate(&client),
		)
		err = app.Start(ctx)
		if err != nil {
			log.Error(err)
		}
		go func() {
			err = client.Run()
			if err != nil {
				log.Error(err)
				buttonStopOnClick(buttonStop)
				ui.MsgBoxError(mainWin, "Error", err.Error())
			}
			client.Shutdown()
		}()
	})
	buttonStop.OnClicked(buttonStopOnClick)
	grid.Append(buttonOpen, 0, 4, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	grid.Append(buttonSave, 1, 4, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)
	grid.Append(buttonRun, 0, 5, 1, 1, true, ui.AlignEnd, true, ui.AlignCenter)
	grid.Append(buttonStop, 1, 5, 1, 1, true, ui.AlignStart, true, ui.AlignCenter)

	labelStatus := ui.NewLabel("relaybaton stopped")
	grid.Append(labelStatus, 0, 6, 2, 1, true, ui.AlignCenter, true, ui.AlignCenter)
	mainWin.Show()
}

func main() {
	err := os.Setenv("GODEBUG", os.Getenv("GODEBUG")+",tls13=1,netdns=go")
	if err != nil {
		log.Fatal(err)
		return
	}
	log.SetReportCaller(true)
	log.SetLevel(log.TraceLevel)

	ui.Main(setupUI)
}

func ApplyConf(conf *config.ConfigGo) {
	ui.QueueMain(func() {
		config.InitLog(conf)
		spinBoxClientPort.SetValue(int(conf.Client.Port))
		entryClientServer.SetText(conf.Client.Server)
		entryClientUsername.SetText(conf.Client.Username)
		passwordEntryClientPassword.SetText(conf.Client.Password)
		radioButtonsClientRoute.SetSelected(clientRoute2UI(conf.Client.ProxyAll))
		comboBoxDNSType.SetSelected(dnsType2UI(conf.DNS.Type))
		entryDNSServer.SetText(conf.DNS.Server)
		entryDNSAddr.SetText(conf.DNS.Addr.String())
		entryLogFile.SetText(conf.Log.File.Name())
		comboBoxLogLevel.SetSelected(int(conf.Log.Level))
	})
}

func GetConf() (*config.ConfigGo, error) {
	var err error
	confTOML := &config.ConfigTOML{
		Log: config.LogTOML{
			File:  entryLogFile.Text(),
			Level: log.Level(comboBoxLogLevel.Selected()).String(),
		},
		DNS: config.DNSToml{
			Type:   dnsType2Conf(comboBoxDNSType.Selected()),
			Server: entryDNSServer.Text(),
			Addr:   entryDNSAddr.Text(),
		},
		Client: config.ClientTOML{
			Port:     spinBoxClientPort.Value(),
			Server:   entryClientServer.Text(),
			Username: entryClientUsername.Text(),
			Password: passwordEntryClientPassword.Text(),
			ProxyAll: clientRoute2Conf(radioButtonsClientRoute.Selected()),
		},
	}
	validate := validator.New()
	conf, err = confTOML.Init()
	if err != nil {
		return nil, err
	}
	err = validate.Struct(confTOML)
	if err != nil {
		return nil, err
	}
	conf.Client, err = confTOML.Client.Init()
	if err != nil {
		return nil, err
	}
	return conf, nil
}

func clientRoute2UI(proxyall bool) int {
	if proxyall {
		return 1
	} else {
		return 0
	}
}

func clientRoute2Conf(proxyall int) bool {
	if proxyall == 0 {
		return false
	} else {
		return true
	}
}

func dnsType2UI(dnsType config.DNSType) int {
	switch dnsType {
	case config.DNSTypeDoH:
		return 2
	case config.DNSTypeDoT:
		return 1
	default:
		return 0
	}
}

func dnsType2Conf(index int) string {
	switch index {
	case 2:
		return "doh"
	case 1:
		return "dot"
	default:
		return "default"
	}
}

func buttonStopOnClick(button *ui.Button) {
	spinBoxClientPort.Enable()
	entryClientServer.Enable()
	entryClientUsername.Enable()
	passwordEntryClientPassword.Enable()
	radioButtonsClientRoute.Enable()
	comboBoxDNSType.Enable()
	entryDNSServer.Enable()
	entryDNSAddr.Enable()
	entryLogFile.Enable()
	comboBoxLogLevel.Enable()
	buttonOpen.Enable()
	buttonSave.Enable()
	buttonRun.Enable()
	buttonStop.Disable()
	cancel()
	err := app.Stop(ctx)
	if err != nil {
		log.Error(err)
		ui.MsgBoxError(mainWin, "Error", err.Error())
	}
}
