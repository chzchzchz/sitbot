package main

import (
	"fmt"
	"image/png"
	"io"
	"os"

	"github.com/chzchzchz/sitbot/ascii"
	"github.com/spf13/cobra"
)

var rootCmd = &cobra.Command{
	Use:   "mkascii",
	Short: "A tool to manipulate ascii and ansi art",
}

var (
	format string
)

func init() {
	rootCmd.PersistentFlags().StringVarP(
		&format, "format", "f", "mirc", "output format (mirc|ansi|png)")
	compileCmd := &cobra.Command{
		Use:   "compile [script]",
		Short: "compile an ascii script from file or stdin",
		Run:   compile,
	}
	rasterizeCmd := &cobra.Command{
		Use:   "rasterize [mircart]",
		Short: "rasterize a mirc art from file or stdin",
		Run:   rasterize,
	}
	rootCmd.AddCommand(compileCmd)
	rootCmd.AddCommand(rasterizeCmd)
}

func main() {
	rootCmd.Execute()
}

func output(a *ascii.ASCII) (err error) {
	switch format {
	case "mirc":
		_, err = os.Stdout.Write(a.Bytes())
	case "ansi":
		_, err = os.Stdout.Write(a.AnsiBytes())
	case "png":
		img, err := ascii.Rasterize(a)
		if err != nil {
			return err
		}
		return png.Encode(os.Stdout, img)
	default:
		return fmt.Errorf("unknown output format %s", format)
	}
	return err
}

func readInput(args []string) (in []byte, err error) {
	f := os.Stdin
	if len(args) > 0 {
		if f, err = os.Open(args[0]); err != nil {
			return nil, err
		}
		defer f.Close()
	}
	in, err = io.ReadAll(f)
	if err != nil && err != io.EOF {
		return nil, err
	}
	return in, nil
}

func rasterize(cmd *cobra.Command, args []string) {
	in, err := readInput(args)
	if err != nil {
		panic(err)
	}
	a, err := ascii.NewASCII(string(in))
	if err != nil {
		panic(err)
	}
	if err := output(a); err != nil {
		panic(err)
	}
}

func compile(cmd *cobra.Command, args []string) {
	in, err := readInput(args)
	if err != nil {
		panic(err)
	}
	g := Grammar{Buffer: string(in), script: Script{groups: make(map[string]*Group)}}
	g.Init()
	if err := g.Parse(); err != nil {
		panic(err)
	}
	g.Execute()
	a, err := run(g.script.stmts)
	if err != nil {
		panic(err)
	}
	if err := output(a); err != nil {
		panic(err)
	}
}
