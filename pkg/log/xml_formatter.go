package log

import (
	"encoding/xml"
	"fmt"

	"github.com/sirupsen/logrus"
)

type XMLFormatter struct {
}

type node struct {
	XMLName  xml.Name
	Time     string `xml:"time,attr,omitempty"`
	Func     string `xml:"func,attr,omitempty"`
	File     string `xml:"file,attr,omitempty"`
	Children []node `xml:",any"`
	Text     string `xml:",chardata"`
}

func (fmter XMLFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	e := node{
		XMLName: xml.Name{
			Local: entry.Level.String(),
		},
		Time: entry.Time.Format("2006-01-02 15:04:05.0000000"),
	}
	if entry.HasCaller() {
		funcVal := entry.Caller.Function
		fileVal := fmt.Sprintf("%s:%d", entry.Caller.File, entry.Caller.Line)
		if funcVal != "" {
			e.Func = funcVal
		}
		if fileVal != "" {
			e.File = fileVal
		}
	}

	if entry.Message != "" {
		elementMsg := node{
			XMLName: xml.Name{
				Local: "msg",
			},
			Text: entry.Message,
		}
		e.Children = append(e.Children, elementMsg)
	}
	for k, v := range entry.Data {
		switch v := v.(type) {
		case error:
			elementMsg := node{
				XMLName: xml.Name{
					Local: k,
				},
				Text: v.Error(),
			}
			e.Children = append(e.Children, elementMsg)
		default:
			elementMsg := node{
				XMLName: xml.Name{
					Local: k,
				},
				Text: fmt.Sprintf("%v", v),
			}
			e.Children = append(e.Children, elementMsg)
		}
	}
	buf, _ := xml.MarshalIndent(e, "", "\t")

	return append(buf, '\n'), nil
}
