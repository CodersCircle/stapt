package main

import (
	"errors"
	"strconv"
	"strings"
)

type ParsedSSH struct {
	Host     string `json:"host"`
	Port     int    `json:"port"`
	Username string `json:"username"`
}

func splitHostPortMaybe(host string) (string, int) {
	host = strings.TrimSpace(strings.Trim(host, "[]"))
	if host == "" {
		return "", 0
	}
	if strings.Count(host, ":") != 1 {
		return host, 0
	}
	i := strings.LastIndex(host, ":")
	p, err := strconv.Atoi(host[i+1:])
	if err != nil || p < 1 || p > 65535 {
		return host, 0
	}
	return host[:i], p
}

func parseSSHCommand(raw string) (ParsedSSH, error) {
	var out ParsedSSH
	out.Port = 22
	s := strings.TrimSpace(raw)
	s = strings.ReplaceAll(s, "\r", " ")
	s = strings.ReplaceAll(s, "\n", " ")
	if s == "" {
		return out, errors.New("ssh command is required")
	}
	parts := strings.Fields(s)
	if len(parts) == 0 {
		return out, errors.New("invalid ssh command")
	}
	if strings.EqualFold(parts[0], "ssh") {
		parts = parts[1:]
	}
	for i := 0; i < len(parts); i++ {
		p := parts[i]
		switch {
		case p == "-p" || p == "-P":
			if i+1 >= len(parts) {
				return out, errors.New("port is missing")
			}
			n, err := strconv.Atoi(parts[i+1])
			if err != nil || n < 1 || n > 65535 {
				return out, errors.New("invalid port")
			}
			out.Port = n
			i++
		case strings.HasPrefix(p, "-p") && len(p) > 2:
			n, err := strconv.Atoi(p[2:])
			if err != nil || n < 1 || n > 65535 {
				return out, errors.New("invalid port")
			}
			out.Port = n
		case p == "-i" || p == "-o" || p == "-F" || p == "-J" || p == "-l":
			if p == "-l" && i+1 < len(parts) {
				out.Username = strings.TrimSpace(parts[i+1])
			}
			if i+1 < len(parts) && !strings.HasPrefix(parts[i+1], "-") {
				i++
			}
		case strings.HasPrefix(p, "-"):
			continue
		default:
			target := strings.Trim(p, `"'`)
			if strings.HasPrefix(strings.ToLower(target), "ssh://") {
				target = target[6:]
			}
			if at := strings.LastIndex(target, "@"); at >= 0 {
				user := target[:at]
				host, port := splitHostPortMaybe(target[at+1:])
				if user != "" {
					out.Username = user
				}
				if host != "" {
					out.Host = host
				}
				if port > 0 {
					out.Port = port
				}
			} else if out.Host == "" {
				host, port := splitHostPortMaybe(target)
				out.Host = host
				if port > 0 {
					out.Port = port
				}
			}
		}
	}
	out.Host = strings.TrimSpace(out.Host)
	out.Username = strings.TrimSpace(out.Username)
	if out.Host == "" {
		return out, errors.New("host is required")
	}
	if out.Username == "" {
		return out, errors.New("username is required")
	}
	if out.Port <= 0 {
		out.Port = 22
	}
	return out, nil
}
