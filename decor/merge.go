package decor

import (
	"fmt"
	"strings"
	"unicode/utf8"
)

// Merge helper func, provides a way to synchronize width of single
// decorator with adjacent decorators of different bar, like so:
//   +--------+---------+
//   |     MERGE(D)     |
//   +--------+---------+
//   |   D1   |   D2    |
//   +--------+---------+
//
func Merge(decorator Decorator, placeholders ...WC) Decorator {
	if _, ok := decorator.Sync(); !ok || len(placeholders) == 0 {
		return decorator
	}
	md := &mergeDecorator{
		Decorator:    decorator,
		placeHolders: make([]*placeHolderDecorator, len(placeholders)),
	}
	md.wc = decorator.SetConfig(md.wc)
	for i, wc := range placeholders {
		wc.Init()
		md.placeHolders[i] = &placeHolderDecorator{
			WC:    wc,
			wsync: make(chan int),
		}
	}
	return md
}

type mergeDecorator struct {
	Decorator
	wc           WC
	placeHolders []*placeHolderDecorator
}

func (d *mergeDecorator) CompoundDecorators() []Decorator {
	decorators := make([]Decorator, len(d.placeHolders)+1)
	decorators[0] = d.Decorator
	for i, ph := range d.placeHolders {
		decorators[i+1] = ph
	}
	return decorators
}

func (md *mergeDecorator) Sync() (chan int, bool) {
	return md.wc.Sync()
}

func (d *mergeDecorator) Decor(st *Statistics) string {
	msg := d.Decorator.Decor(st)
	msgLen := utf8.RuneCountInString(msg)
	pWidth := msgLen / (len(d.placeHolders) + 1)
	mod := msgLen % (len(d.placeHolders) + 1)
	d.wc.wsync <- pWidth + mod
	for _, ph := range d.placeHolders {
		ph.wsync <- pWidth
	}
	max := <-d.wc.wsync
	for _, ph := range d.placeHolders {
		max += <-ph.wsync
	}
	if (d.wc.C & DextraSpace) != 0 {
		max++
	}
	return fmt.Sprintf(fmt.Sprintf(d.wc.dynFormat, max), msg)
}

type placeHolderDecorator struct {
	WC
	wsync chan int
}

func (d *placeHolderDecorator) Decor(st *Statistics) string {
	go func() {
		width := <-d.wsync
		msg := strings.Repeat(" ", width)
		d.wsync <- utf8.RuneCountInString(d.FormatMsg(msg))
	}()
	return ""
}
