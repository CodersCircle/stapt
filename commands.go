package main

import (
	"errors"
	"strings"
)

type QuickCommand struct {
	Key         string `json:"key"`
	Group       string `json:"group"`
	Label       string `json:"label"`
	Destructive bool   `json:"destructive"`
}

type CommandResult struct {
	OK     bool   `json:"ok"`
	Output string `json:"output"`
	Error  string `json:"error"`
}

func commandsForTech(tech string) []QuickCommand {
	set := map[string]bool{}
	for _, t := range strings.Split(tech, ",") {
		t = strings.TrimSpace(strings.ToLower(t))
		if t != "" {
			set[t] = true
		}
	}
	if len(set) == 0 {
		set["php"] = true
		set["git"] = true
	}
	var out []QuickCommand
	add := func(items ...QuickCommand) { out = append(out, items...) }
	if set["php"] || set["laravel"] || set["composer"] {
		add(
			QuickCommand{Key: "php.version", Group: "PHP", Label: "PHP Version"},
			QuickCommand{Key: "php.modules", Group: "PHP", Label: "PHP Modules"},
		)
	}
	if set["laravel"] {
		add(
			QuickCommand{Key: "artisan.version", Group: "Laravel", Label: "Artisan Version"},
			QuickCommand{Key: "artisan.optimize_clear", Group: "Laravel", Label: "Optimize Clear", Destructive: true},
			QuickCommand{Key: "artisan.config_cache", Group: "Laravel", Label: "Config Cache"},
			QuickCommand{Key: "artisan.route_cache", Group: "Laravel", Label: "Route Cache"},
			QuickCommand{Key: "artisan.view_cache", Group: "Laravel", Label: "View Cache"},
			QuickCommand{Key: "artisan.migrate_status", Group: "Laravel", Label: "Migration Status"},
			QuickCommand{Key: "artisan.migrate", Group: "Laravel", Label: "Migrate", Destructive: true},
			QuickCommand{Key: "artisan.storage_link", Group: "Laravel", Label: "Storage Link"},
		)
	}
	if set["composer"] || set["php"] || set["laravel"] {
		add(
			QuickCommand{Key: "composer.version", Group: "Composer", Label: "Version"},
			QuickCommand{Key: "composer.install", Group: "Composer", Label: "Install"},
			QuickCommand{Key: "composer.dump", Group: "Composer", Label: "Dump Autoload"},
		)
	}
	if set["node"] {
		add(
			QuickCommand{Key: "node.version", Group: "Node/NPM", Label: "Node Version"},
			QuickCommand{Key: "npm.version", Group: "Node/NPM", Label: "NPM Version"},
			QuickCommand{Key: "npm.install", Group: "Node/NPM", Label: "Install"},
			QuickCommand{Key: "npm.build", Group: "Node/NPM", Label: "Build"},
		)
	}
	if set["git"] {
		add(
			QuickCommand{Key: "git.version", Group: "Git", Label: "Version"},
			QuickCommand{Key: "git.status", Group: "Git", Label: "Status"},
			QuickCommand{Key: "git.branch", Group: "Git", Label: "Branch"},
			QuickCommand{Key: "git.log", Group: "Git", Label: "Log"},
			QuickCommand{Key: "git.pull", Group: "Git", Label: "Pull", Destructive: true},
		)
	}
	if set["python"] {
		add(
			QuickCommand{Key: "python.version", Group: "Python", Label: "Version"},
			QuickCommand{Key: "pip.version", Group: "Python", Label: "Pip Version"},
		)
	}
	if set["go"] {
		add(QuickCommand{Key: "go.version", Group: "Go", Label: "Version"})
	}
	if set["ruby"] {
		add(QuickCommand{Key: "ruby.version", Group: "Ruby", Label: "Version"})
	}
	return out
}

func commandLine(key string) (string, bool, error) {
	destructive := false
	var cmd string
	switch key {
	case "php.version":
		cmd = "php -v"
	case "php.modules":
		cmd = "php -m"
	case "artisan.version":
		cmd = "php artisan --version"
	case "artisan.optimize_clear":
		cmd, destructive = "php artisan optimize:clear", true
	case "artisan.config_cache":
		cmd = "php artisan config:cache"
	case "artisan.route_cache":
		cmd = "php artisan route:cache"
	case "artisan.view_cache":
		cmd = "php artisan view:cache"
	case "artisan.migrate_status":
		cmd = "php artisan migrate:status"
	case "artisan.migrate":
		cmd, destructive = "php artisan migrate --force", true
	case "artisan.storage_link":
		cmd = "php artisan storage:link"
	case "composer.version":
		cmd = "composer --version"
	case "composer.install":
		cmd = "composer install --no-interaction"
	case "composer.dump":
		cmd = "composer dump-autoload --no-interaction"
	case "node.version":
		cmd = "node -v"
	case "npm.version":
		cmd = "npm -v"
	case "npm.install":
		cmd = "npm install"
	case "npm.build":
		cmd = "npm run build"
	case "git.version":
		cmd = "git --version"
	case "git.status":
		cmd = "git status -sb"
	case "git.branch":
		cmd = "git branch -vv"
	case "git.log":
		cmd = "git log -5 --oneline"
	case "git.pull":
		cmd, destructive = "git pull --ff-only", true
	case "python.version":
		cmd = "python3 --version || python --version"
	case "pip.version":
		cmd = "pip3 --version || pip --version"
	case "go.version":
		cmd = "go version"
	case "ruby.version":
		cmd = "ruby --version"
	default:
		return "", false, errors.New("unknown command")
	}
	return cmd, destructive, nil
}
