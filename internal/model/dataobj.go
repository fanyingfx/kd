package model

import (
	"fmt"
	"regexp"
	"strings"
	"testing"

	"github.com/Karmenzind/kd/pkg"
	d "github.com/Karmenzind/kd/pkg/decorate"
	"go.uber.org/zap"
)

type CollinsItem struct {
	Additional   string     `json:"a"`
	MajorTrans   string     `json:"maj"`
	ExampleLists [][]string `json:"eg"`
	// MajorTransCh string // 备用
}

type Result struct {
	Found      bool   `json:"-"`
	Prompt     string `json:"-"`
	IsEN       bool   `json:"-"`
	IsPhrase   bool   `json:"-"`
	IsLongText bool   `json:"-"`
	Query      string `json:"-"`

	Keyword    string            `json:"k"`
	Pronounce  map[string]string `json:"pron"`
	Paraphrase []string          `json:"para"`
	// UpdateTime time.Time
	// CreateTime time.Time

	Examples map[string][][]string `json:"eg"`

	// XXX (k): <2023-11-15> 直接提到第一级
	Collins struct {
		Star              int    `json:"star"`
		ViaRank           string `json:"rank"`
		AdditionalPattern string `json:"pat"`

		Items []*CollinsItem `json:"li"`
	} `json:"co"`

	Output  string   `json:"-"`
	History chan int `json:"-"`
}

func (r *Result) Initialize() {
	if m, e := regexp.MatchString("^[A-Za-z0-9 -.?]+$", r.Query); e == nil && m {
		r.IsEN = true
		if strings.Contains(r.Query, " ") {
			r.IsPhrase = true
		}
		zap.S().Debugf("Query: isEn: %v isPhrase: %v\n", r.IsEN, r.IsPhrase)
	}
}

// func emojifyIfNeeded(str string, enableEmoji bool) {
// }

// func (r *Result) ToQueryDaemonJSON() ([]byte, error) {
//     q := model.QueryDaemon{
//     }
//     return json.Marshal(r)
// }

// func (r *Result) FromDaemonResponseJSON() []byte {
// }

var p = regexp.MustCompile("^([^\u4e00-\u9fa5]+) ([^ ]*[\u4e00-\u9fa5]+.*)$")
var normalSentence = regexp.MustCompile("^[A-Za-z]+ ")

// XXX
func cutCollinsTrans(line string) (string, string) {
	g := p.FindStringSubmatch(line)
	if len(g) == 3 {
		return g[1], g[2]
	}
	return "", ""
}

func (r *Result) PrettyFormat(onlyEN bool) string {
	egPref := d.EgPref("≫  ")
	if r.Output != "" {
		return r.Output
	}
	s := []string{}

	var title string
	if r.Keyword == "" {
		title = r.Query
	} else {
		title = r.Keyword
	}

	header := d.Title(title)
	// s = append(s, d.Title(title))

	pronStr := ""
	for nation, v := range r.Pronounce {
		// pronStr += d.Na(nation) + d.Pron(v)
		v = strings.Trim(v, "[]")
		pronStr += fmt.Sprintf("%s %s / ", nation, v)
	}
	if pronStr != "" {
		pronStr = d.Pron(fmt.Sprintf("[%s]", strings.Trim(pronStr, "/ ")))
		header = fmt.Sprintf("%s    %s", header, pronStr)
	}
	s = append(s, header)

	// TODO wth is de morgan's law
	if !(onlyEN && r.IsEN) {
		for _, para := range r.Paraphrase {
			if para == "" {
				// FIXME (k): <2023-12-15> 从收集步骤规避
				continue
			}
			if normalSentence.MatchString(para) {
				s = append(s, d.Para(para))
			} else {
				splited := strings.SplitN(para, " ", 2)
				if len(splited) == 2 {
					s = append(s, fmt.Sprintf("%s %s", d.Property(splited[0]), d.Para(splited[1])))
				} else {
					s = append(s, d.Para(para))
				}
			}
		}
	}

	// cutoff := strings.Repeat("–", cutoffLength())
	cutoff := strings.Repeat("⸺", cutoffLength())

	rankParts := []string{}
	if r.Collins.Star > 0 {
		rankParts = append(rankParts, d.Star(strings.Repeat("★", r.Collins.Star)))
	}
	if r.Collins.ViaRank != "" {
		rankParts = append(rankParts, d.Rank(r.Collins.ViaRank))
	}
	if r.Collins.AdditionalPattern != "" {
		rankParts = append(rankParts, d.Rank(r.Collins.AdditionalPattern))
	}
	if len(rankParts) > 0 {
		s = append(s, strings.Join(rankParts, " "))
	}

	if r.IsEN && len(r.Collins.Items) > 0 {
		s = append(s, d.Line(cutoff))
		for idx, i := range r.Collins.Items {
			var transExpr string
			if onlyEN {
				transExpr, _ = cutCollinsTrans(i.MajorTrans)
				if transExpr == "" {
					transExpr = i.MajorTrans
				}
			} else {
				transExpr = i.MajorTrans
			}

			var piece string
			piece = fmt.Sprintf("%s. ", d.Idx(idx+1))
			if i.Additional != "" {
				if strings.HasPrefix(i.Additional, "[") && strings.HasSuffix(i.Additional, "]") {
					piece += d.Addi(i.Additional + " ")
				} else {
					piece += d.Addi("(" + i.Additional + ") ")
				}
			}
			piece += d.CollinsPara(transExpr)
			s = append(s, piece)

			for _, ePair := range i.ExampleLists {
				var eRepr string
				if onlyEN {
					eRepr = ePair[0]
				} else {
					eRepr = strings.Join(ePair, "  ")
				}
				s = append(s, fmt.Sprintf("   %s %s", egPref, d.Eg(eRepr)))
				// s = append(s, d.Eg(fmt.Sprintf("   e.g. %s", eRepr)))
			}
		}
	}

	if (!r.IsEN || (r.IsEN && len(r.Collins.Items) == 0)) && len(r.Examples) > 0 {
		s = append(s, d.Line(cutoff))
		for _, tab := range []string{"bi", "or"} {
			if exampleList, ok := r.Examples[tab]; ok {
				for _, item := range exampleList {
					if p := displayExample(item, tab, onlyEN, r.IsEN); p != "" {
						// s = append(s, fmt.Sprintf("%d. %s", idx+1, p))
						s = append(s, fmt.Sprintf("%s %s", egPref, p))
					}
				}
				break
			}
		}
	}

	// s = append(s, r.Pronounce)
	r.Output = strings.Join(s, "\n")
	return r.Output
}

func displayExample(item []string, tab string, onlyEN bool, isEN bool) string {
	var r string
	switch tab {
	case "bi":
		if onlyEN {
			r = d.EgEn(item[0])
		} else {
			r = fmt.Sprintf("%s %s", d.EgEn(item[0]), d.EgCh(item[1]))
		}
	case "au":
		// TODO 增加来源渲染
		r = fmt.Sprintf("%s (%s)", d.EgEn(item[0]), d.EgCh(item[1]))
	case "or":
		if onlyEN {
			if isEN {
				r = d.EgEn(item[0])
			} else {
				r = d.EgEn(item[1])
			}
		} else {
			r = fmt.Sprintf("%s %s", d.EgEn(item[0]), d.EgCh(item[1]))
		}
	}
	return r
}

func cutoffLength() int {
	width, _, err := pkg.GetTermSize()
	if err != nil {
		width = 44
	}
	return width - 2
}

func TestPrint(t *testing.T) {
}
