package admin

import (
	"bufio"
	"context"
	"fmt"
	_node "github.com/cryptopunkscc/astrald/node"
	"io"
	"strings"
)

const promptString = "> "

type cmdFunc func(io.ReadWriter, *_node.Node, []string) error
type cmdMap map[string]cmdFunc

var commands cmdMap

func init() {
	commands = cmdMap{
		"help":     help,
		"peers":    peers,
		"link":     link,
		"contacts": cmdContacts,
		"info":     info,
		"parse":    parse,
		"add":      add,
		"present":  present,
	}
	_ = _node.RegisterService("admin", listen)
}

func help(stream io.ReadWriter, _ *_node.Node, _ []string) error {
	fmt.Fprintf(stream, "commands:")
	for k := range commands {
		fmt.Fprintf(stream, " %s", k)
	}
	fmt.Fprintf(stream, "\n")

	return nil
}

func serve(stream io.ReadWriteCloser, node *_node.Node) {
	defer stream.Close()

	prompt := node.Alias() + promptString

	scanner := bufio.NewScanner(stream)
	stream.Write([]byte(prompt))

	for scanner.Scan() {
		words := strings.Split(scanner.Text(), " ")
		if len(words) == 0 {
			continue
		}

		cmd, args := words[0], words[1:]

		fn, ok := commands[cmd]
		if ok {
			err := fn(stream, node, args)
			if err != nil {
				fmt.Fprintf(stream, "error: %v\n", err)
			} else {
				fmt.Fprintf(stream, "ok\n")
			}
		} else {
			fmt.Fprintf(stream, "no such command\n")
		}
		stream.Write([]byte(prompt))
	}
}

func listen(ctx context.Context, node *_node.Node) error {
	port, err := node.Ports.RegisterContext(ctx, "admin")
	if err != nil {
		return err
	}

	for req := range port.Queries() {
		// Only accept local requests
		if !req.Caller().IsEqual(node.Identity()) {
			req.Reject()
			continue
		}
		conn := req.Accept()

		go serve(conn, node)
	}

	return nil
}
