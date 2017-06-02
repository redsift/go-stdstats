package stdstats

import (
	"fmt"
	"image"
	"image/color"
	"image/png"
	"os"
	"sort"
	"strconv"
	"strings"
	"time"

	svg "github.com/ajstarks/svgo"
	"github.com/redsift/go-errs"
	"github.com/redsift/go-stats"
	"github.com/vdobler/chart"
	"github.com/vdobler/chart/imgg"
	"github.com/vdobler/chart/svgg"
	"github.com/vdobler/chart/txtg"
)

type dataList map[string]map[string][]float64

const (
	initialListLength = 1024
	width             = 100
	height            = 40
	initialSourceSize = 32
)

// style taken from http://design.sunlightlabs.com/projects/Sunlight-StyleGuide-DataViz.pdf for now
const fontFamilyTitle = "Consolas, Menlo, Monaco, monospace, serif"

var colourMain = color.RGBA{0x63, 0x5F, 0x5D, 0xff}
var colourBackground = color.RGBA{0xff, 0xff, 0xff, 0xff}
var colourBackgroundAlt = color.RGBA{0xef, 0xec, 0xea, 0xff}

var colourSeriesL = []color.RGBA{
	color.RGBA{0x33, 0xB6, 0xD0, 0xff}, //1
	color.RGBA{0xF2, 0xDA, 0x57, 0xff}, //2
	color.RGBA{0xB3, 0x96, 0xAD, 0xff}, //3
	color.RGBA{0x7A, 0xBF, 0xCC, 0xff}, //4
	color.RGBA{0xF6, 0xB6, 0x56, 0xff}, //5
	color.RGBA{0xE2, 0x5A, 0x42, 0xff}, //6
	color.RGBA{0xA0, 0xB7, 0x00, 0xff}, //7
	color.RGBA{0xDC, 0xBD, 0xCF, 0xff}, //8
	color.RGBA{0xC8, 0xD7, 0xA1, 0xff}, //9
	color.RGBA{0xB0, 0xCB, 0xDB, 0xff}, //10
}

var colourSeriesM = []color.RGBA{
	color.RGBA{0x42, 0xA5, 0xB3, 0xff}, //1
	color.RGBA{0xE3, 0xBA, 0x22, 0xff}, //2
	color.RGBA{0x8E, 0x6C, 0x8A, 0xff}, //3
	color.RGBA{0x0F, 0x8C, 0x79, 0xff}, //4
	color.RGBA{0xE5, 0x84, 0x29, 0xff}, //5
	color.RGBA{0xBD, 0x2D, 0x28, 0xff}, //6
	color.RGBA{0x5C, 0x81, 0x00, 0xff}, //10
	color.RGBA{0xD1, 0x5A, 0x86, 0xff}, //7
	color.RGBA{0x6B, 0x99, 0xA1, 0xff}, //8
	color.RGBA{0x6B, 0xBB, 0xA1, 0xff}, //9
}

func colourForIndex(i int) (color.RGBA, color.RGBA) {
	d := i % len(colourSeriesL)

	return colourSeriesL[d], colourSeriesM[d]
}

// simple local collector for SDK
type stdoutC struct {
	dump      string
	hist      dataList
	source    []string
	whitelist []string
}

func NewStdout(dump string, whitelist []string) stats.Collector {
	return &stdoutC{dump, make(dataList), make([]string, 0, initialSourceSize), whitelist}
}

func (d *stdoutC) whiteListed(stat string) bool {
	if len(d.whitelist) == 0 {
		return true
	}

	for _, s := range d.whitelist {
		if s == stat {
			return true
		}
	}
	return false
}

func (d *stdoutC) Inform(title, text string, tags []string) {}

func (d *stdoutC) Error(pe *errs.PropagatedError, tags []string) {}

func (d *stdoutC) Count(stat string, count float64, tags []string) {
	//TODO: this
}

func (d *stdoutC) Gauge(stat string, value float64, tags []string) {
	//TODO: this
}

func (d *stdoutC) Timing(stat string, value time.Duration, tags []string) {
	if !d.whiteListed(stat) {
		return
	}

	tg := strings.Join(tags, ",")

	fst, exists := d.hist[stat]
	if !exists {
		fst = make(map[string][]float64)
		d.source = append(d.source, stat)
	}

	data, exists := fst[tg]
	if !exists {
		data = make([]float64, 0, initialListLength)
	}
	v := float64(value) / float64(time.Millisecond)
	if v < 1 {
		// do this so the charting puts these in the 0 bucket for the histogram
		v = 0.0
	}

	data = append(data, v)
	fst[tg] = data
	d.hist[stat] = fst
}

func (d *stdoutC) Histogram(stat string, value float64, tags []string) {

}

func (d *stdoutC) renderToStdout(hists []*chart.HistChart) {

	fmt.Println(strings.Repeat("-", width))
	fmt.Println()

	for _, hist := range hists {
		tgr := txtg.New(width, height)
		hist.Plot(tgr)
		fmt.Println(tgr.String())
	}
}

func (d *stdoutC) renderToSVGFile(name string, hists []*chart.HistChart) {
	svgFile, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer svgFile.Close()

	w := width * 10
	h := height * 10

	svg := svg.New(svgFile)
	svg.Start(w, h*len(hists))
	svg.Title(name)
	svg.Rect(0, 0, w, h*len(hists), "fill: #ffffff")

	for i, hist := range hists {
		sgr := svgg.AddTo(svg, 0, i*h, w, h, "", 12, colourBackground)
		hist.Plot(sgr)
	}

	svg.End()
}

func (d *stdoutC) renderToPNGFile(name string, hists []*chart.HistChart) {
	imgFile, err := os.Create(name)
	if err != nil {
		panic(err)
	}
	defer imgFile.Close()

	w := width * 10
	h := height * 10

	img := image.NewRGBA(image.Rect(0, 0, w, h*len(hists)))

	for i, hist := range hists {
		igr := imgg.AddTo(img, 0, i*h, w, h, colourBackground, nil, nil)
		hist.Plot(igr)
	}
	png.Encode(imgFile, img)
}

func sortTags(data map[string][]float64) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func (d *stdoutC) Close() {
	if len(d.hist) == 0 {
		return
	}

	test := strings.ToLower(d.dump)
	ascii := test == "" || test == "stdout"

	hists := make([]*chart.HistChart, len(d.hist))

	for i, k := range d.source {
		tg := d.hist[k]
		// fmt.Println(k, tg)

		hist := chart.HistChart{Title: k, Stacked: false, Counts: true, Shifted: true}
		hist.YRange.MaxMode.Expand = chart.ExpandTight
		hist.YRange.Label = "Frequency [count]"
		hist.YRange.TicSetting.Format = func(v float64) string {
			return strconv.FormatFloat(v, 'f', 0, 64)
		}

		hist.XRange.MinMode.Lower = 0.0
		hist.XRange.MinMode.Expand = chart.ExpandTight
		hist.XRange.MinMode.Constrained = true
		hist.XRange.Label = "Time [s]"
		hist.XRange.TicSetting.Format = func(v float64) string {
			if v < 500 {
				return strconv.FormatFloat(v, 'f', 1, 64) + "ms"
			}
			if v < 1100 {
				return strconv.FormatFloat(v, 'f', 0, 64) + "ms"
			}

			return strconv.FormatFloat((v / 1000), 'f', 2, 64)
		}

		hist.Key.Pos = "ort"
		hist.Key.Cols = 1

		titleFont := chart.Font{Color: colourMain, Name: fontFamilyTitle}
		keyFont := chart.Font{Color: colourMain, Name: fontFamilyTitle, Size: chart.TinyFontSize}

		hist.Options = chart.PlotOptions{
			chart.TitleElement: chart.Style{Font: titleFont},
			chart.KeyElement:   chart.Style{Font: keyFont},
		}

		for i, s := range sortTags(tg) {
			v := tg[s]
			style := chart.Style{}

			if !ascii {
				fill, line := colourForIndex(i)
				style.FillColor = fill
				style.LineColor = line
				style.LineWidth = 1
			}
			hist.AddData(s, v, style)
		}

		hists[i] = &hist
	}

	switch {
	case ascii:
		d.renderToStdout(hists)
	case strings.HasSuffix(test, ".png"):
		d.renderToPNGFile(d.dump, hists)
	default:
		d.renderToSVGFile(d.dump, hists)
	}
}
