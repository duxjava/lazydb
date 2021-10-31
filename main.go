// Copyright 2014 The gocui Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

package main

import (
	"fmt"
	"log"
	"strconv"
	"strings"

	"database/sql"

	_ "github.com/go-sql-driver/mysql"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
	"github.com/jroimartin/gocui"
)

var (
	viewArr = []string{"db", "tables", "output", "where"}
	active  = 0
)


func setCurrentViewOnTop(g *gocui.Gui, name string) (*gocui.View, error) {
	if _, err := g.SetCurrentView(name); err != nil {
		return nil, err
	}
	return g.SetViewOnTop(name)
}

func nextView(g *gocui.Gui, v *gocui.View) error {
	nextIndex := (active + 1) % len(viewArr)
	name := viewArr[nextIndex]

	if _, err := setCurrentViewOnTop(g, name); err != nil {
		return err
	}

    if nextIndex == 0 || nextIndex == 2 || nextIndex == 3 {
		g.Cursor = true
	} else {
		g.Cursor = false
	}

	active = nextIndex
	return nil
}

func cursorDown(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()

        numlines := strings.Count(v.Buffer(), "\n") - 2
        if numlines != cy {
            if err := v.SetCursor(cx, cy+1); err != nil {
                ox, oy := v.Origin()
                if err := v.SetOrigin(ox, oy+1); err != nil {
                    return err
                }
            }
        }
	}
	return nil
}

func cursorUp(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()
		if err := v.SetCursor(cx, cy-1); err != nil && oy > 0 {
			if err := v.SetOrigin(ox, oy-1); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorLeft(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		cx, cy := v.Cursor()
		ox, oy := v.Origin()
		if err := v.SetCursor(cx-1, cy); err != nil && ox > 0 {
			if err := v.SetOrigin(ox-1, oy); err != nil {
				return err
			}
		}
	}
	return nil
}

func cursorRight(g *gocui.Gui, v *gocui.View) error {
	if v != nil {
		ox, oy := v.Origin()
		cx, cy := v.Cursor()

        numlines := len(strings.Split(v.Buffer(), "\n")[0]) - 1

        if numlines != ox {
		    if err := v.SetCursor(cx+1, cy); err != nil {
			    if err := v.SetOrigin(ox+1, oy); err != nil {
				    return err
			    }
		    }
        }
	}
	return nil
}

func refreshDb(g *gocui.Gui, msg *gocui.View) error {

    dbView, _ := g.View("db")
    dbName, _ := dbView.Line(0)
    db, _ := sql.Open("mysql", dbName)

    defer db.Close()

    results, _ := db.Query("SHOW TABLES")

    if v, err := g.View("tables"); err == nil {
        v.Clear()
        for results.Next() {
            var name string
            err = results.Scan(&name)
            if err != nil {
                panic(err.Error())
            }

		    fmt.Fprintln(v, name)
        }
    }

	return nil
}

func outputDb(g *gocui.Gui, msg *gocui.View) error {

    dbView, _ := g.View("db")
    dbName, _ := dbView.Line(0)
    db, _ := sql.Open("mysql", dbName)

    defer db.Close()

    whereView, _ := g.View("where")
    whereName, _ := whereView.Line(0)

    tableView, _ := g.View("tables")
    _, cy := tableView.Cursor()
    tableName, _ := tableView.Line(cy)

    limitView, _ := g.View("limit")
    limitName, _ := limitView.Line(0)

    rows, _ := db.Query(fmt.Sprintf("select * from %s where %s order by id limit %s", tableName, whereName, limitName))

    defer rows.Close()
    columns, _ := rows.Columns()

    count := len(columns)
    values := make([]interface{}, count)
    valuePtrs := make([]interface{}, count)

    if v, err := g.View("output"); err == nil {
        v.Clear()
        tableHeader := table.Row{}
        outputC := make(map[string]bool)

        t := table.NewWriter()
        t.Style().Format.Header = text.FormatDefault

        for rows.Next() {
            for i := 0; i < count; i++ {
              valuePtrs[i] = &values[i]
            }

            rows.Scan(valuePtrs...)
            var row = table.Row{}

            for i, col := range columns {
                val := values[i]

                var b = "null"
                if val != nil {
                    b = string(val.([]byte))
                }
                row = append(row, b)

                if (!outputC[string(col)]) {
                    tableHeader = append(tableHeader, string(col))
                    outputC[string(col)] = true
                }

            }

            t.AppendRow(row)
            t.AppendSeparator()
        }

        t.AppendHeader(tableHeader)
		fmt.Fprintln(v, t.Render())
    }

	return nil
}

func selectAll(g *gocui.Gui, v *gocui.View) error {

    dbView, _ := g.View("db")
    dbName, _ := dbView.Line(0)
    db, _ := sql.Open("mysql", dbName)

    defer db.Close()

    _, cy := v.Cursor()
    dbName, _ = v.Line(cy)

    limitView, _ := g.View("limit")
    limitName, _ := limitView.Line(0)

    rows, _ := db.Query(fmt.Sprintf("select * from %s limit %s", dbName, limitName))
    defer rows.Close()
    columns, _ := rows.Columns()

    count := len(columns)
    values := make([]interface{}, count)
    valuePtrs := make([]interface{}, count)

    if v, err := g.View("output"); err == nil {
        v.Clear()
        tableHeader := table.Row{}
        outputC := make(map[string]bool)

        t := table.NewWriter()
        t.Style().Format.Header = text.FormatDefault

        for rows.Next() {
            for i := 0; i < count; i++ {
              valuePtrs[i] = &values[i]
            }

            rows.Scan(valuePtrs...)
            var row = table.Row{}

            for i, col := range columns {
                val := values[i]

                var b = "null"
                if val != nil {
                    b = string(val.([]byte))
                }
                row = append(row, b)

                if (!outputC[string(col)]) {
                    tableHeader = append(tableHeader, string(col))
                    outputC[string(col)] = true
                }

            }

            t.AppendRow(row)
            t.AppendSeparator()
        }

        t.AppendHeader(tableHeader)
		fmt.Fprintln(v, t.Render())
    }

	return nil
}

func nexPage(g *gocui.Gui, v *gocui.View) error {

    dbView, _ := g.View("db")
    dbName, _ := dbView.Line(0)
    db, _ := sql.Open("mysql", dbName)

    defer db.Close()

    tableView, _ := g.View("tables")
    _, cy := tableView.Cursor()
    tableName, _ := tableView.Line(cy)

    limitView, _ := g.View("limit")
    limitName, _ := limitView.Line(0)

    offsetView, _ := g.View("offset")
    offsetLine, _ := offsetView.Line(0)

    limit, _ := strconv.Atoi(limitName)
    offset, _ := strconv.Atoi(offsetLine)
    offset += 1
    offsetView.Clear()
	fmt.Fprint(offsetView, offset)

    offset = (limit * offset) - limit;

    rows, err := db.Query(fmt.Sprintf("select * from %s limit %s OFFSET %d", tableName, limitName, offset))
    if err != nil {
        panic(err)
    }
    defer rows.Close()
    columns, _ := rows.Columns()

    count := len(columns)
    values := make([]interface{}, count)
    valuePtrs := make([]interface{}, count)

    if v, err := g.View("output"); err == nil {
        v.Clear()
        tableHeader := table.Row{}
        outputC := make(map[string]bool)

        t := table.NewWriter()
        t.Style().Format.Header = text.FormatDefault

        for rows.Next() {
            for i := 0; i < count; i++ {
              valuePtrs[i] = &values[i]
            }

            rows.Scan(valuePtrs...)
            var row = table.Row{}

            for i, col := range columns {
                val := values[i]

                var b = "null"
                if val != nil {
                    b = string(val.([]byte))
                }
                row = append(row, b)

                if (!outputC[string(col)]) {
                    tableHeader = append(tableHeader, string(col))
                    outputC[string(col)] = true
                }

            }

            t.AppendRow(row)
            t.AppendSeparator()
        }

        t.AppendHeader(tableHeader)
		fmt.Fprintln(v, t.Render())
    }

	return nil
}

func prevPage(g *gocui.Gui, v *gocui.View) error {

    dbView, _ := g.View("db")
    dbName, _ := dbView.Line(0)
    db, _ := sql.Open("mysql", dbName)

    defer db.Close()

    tableView, _ := g.View("tables")
    _, cy := tableView.Cursor()
    tableName, _ := tableView.Line(cy)

    limitView, _ := g.View("limit")
    limitName, _ := limitView.Line(0)

    offsetView, _ := g.View("offset")
    offsetLine, _ := offsetView.Line(0)

    limit, _ := strconv.Atoi(limitName)
    offset, _ := strconv.Atoi(offsetLine)

    if offset > 1 {
        offset -= 1
    }
    offsetView.Clear()
	fmt.Fprint(offsetView, offset)
    offset = (limit * offset) - limit;

    rows, err := db.Query(fmt.Sprintf("SELECT * FROM %s LIMIT %s OFFSET %d", tableName, limitName, offset))
    if err != nil {
        panic(err)
    }
    defer rows.Close()
    columns, _ := rows.Columns()

    count := len(columns)
    values := make([]interface{}, count)
    valuePtrs := make([]interface{}, count)

    if v, err := g.View("output"); err == nil {
        v.Clear()
        tableHeader := table.Row{}
        outputC := make(map[string]bool)

        t := table.NewWriter()
        t.Style().Format.Header = text.FormatDefault

        for rows.Next() {
            for i := 0; i < count; i++ {
              valuePtrs[i] = &values[i]
            }

            rows.Scan(valuePtrs...)
            var row = table.Row{}

            for i, col := range columns {
                val := values[i]

                var b = "null"
                if val != nil {
                    b = string(val.([]byte))
                }
                row = append(row, b)

                if (!outputC[string(col)]) {
                    tableHeader = append(tableHeader, string(col))
                    outputC[string(col)] = true
                }

            }

            t.AppendRow(row)
            t.AppendSeparator()
        }

        t.AppendHeader(tableHeader)
		fmt.Fprintln(v, t.Render())
    }

	return nil
}

func quit(g *gocui.Gui, v *gocui.View) error {
	return gocui.ErrQuit
}

func keybindings(g *gocui.Gui) error {
    if err := g.SetKeybinding("", gocui.KeyCtrlC, gocui.ModNone, quit); err != nil {
		return err
	}
	if err := g.SetKeybinding("", gocui.KeyCtrlR, gocui.ModNone, refreshDb); err != nil {
		return err
	}
    if err := g.SetKeybinding("", gocui.KeyTab, gocui.ModNone, nextView); err != nil {
		return err
	}

	if err := g.SetKeybinding("tables", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("tables", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
    if err := g.SetKeybinding("tables", gocui.KeyEnter, gocui.ModNone, selectAll); err != nil {
		return err
	}
    if err := g.SetKeybinding("tables", gocui.KeyEnter, gocui.ModNone, selectAll); err != nil {
		return err
	}

    if err := g.SetKeybinding("output", gocui.KeyArrowDown, gocui.ModNone, cursorDown); err != nil {
		return err
	}
	if err := g.SetKeybinding("output", gocui.KeyArrowUp, gocui.ModNone, cursorUp); err != nil {
		return err
	}
    if err := g.SetKeybinding("output", gocui.KeyArrowRight, gocui.ModNone, cursorRight); err != nil {
		return err
	}
	if err := g.SetKeybinding("output", gocui.KeyArrowLeft, gocui.ModNone, cursorLeft); err != nil {
		return err
	}
	if err := g.SetKeybinding("output", gocui.KeyCtrlN, gocui.ModNone, nexPage); err != nil {
		return err
	}
	if err := g.SetKeybinding("output", gocui.KeyCtrlB, gocui.ModNone, prevPage); err != nil {
		return err
	}

	if err := g.SetKeybinding("where", gocui.KeyEnter, gocui.ModNone, outputDb); err != nil {
		return err
	}

	return nil
}

func layout(g *gocui.Gui) error {
	maxX, maxY := g.Size()

	if v, err := g.SetView("db", 0, 0, 30, 2); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Editable = true
        v.Title = "DB"
        fmt.Fprint(v, "sail:password@/agc")
        if _, err := g.SetCurrentView("db"); err != nil {
		    return err
	    }
	}

	if v, err := g.SetView("tables", 0, 3, 30, maxY - 1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
		v.Highlight = true
		v.SelBgColor = gocui.ColorGreen
		v.SelFgColor = gocui.ColorBlack
        v.Title = "Tables"
	}

	if v, err := g.SetView("output", 31, 0, maxX - 1, maxY - 4); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}
        v.Title = "Output"
	}

	if v, err := g.SetView("where", 31, maxY - 3, maxX - 22, maxY - 1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.Wrap = true
        v.Title = "Where"
	}

    if v, err := g.SetView("limit", maxX - 21, maxY - 3, maxX - 11, maxY - 1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.Wrap = true
        v.Title = "Limit"
        fmt.Fprint(v, "30")
	}

    if v, err := g.SetView("offset", maxX - 10, maxY - 3, maxX - 1, maxY - 1); err != nil {
		if err != gocui.ErrUnknownView {
			return err
		}

		v.Editable = true
		v.Wrap = true
        v.Title = "Offset"
        fmt.Fprint(v, "1")
	}

	return nil
}

func main() {
	g, err := gocui.NewGui(gocui.OutputNormal)

	if err != nil {
		log.Panicln(err)
	}

	defer g.Close()

    g.Highlight = true
	g.Cursor = true
	g.SelFgColor = gocui.ColorGreen

	g.SetManagerFunc(layout)

	if err := keybindings(g); err != nil {
		log.Panicln(err)
	}

	if err := g.MainLoop(); err != nil && err != gocui.ErrQuit {
		log.Panicln(err)
	}
}
