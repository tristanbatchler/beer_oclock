# Beer o'clock

## Quick start

### Install required tools
1. Initialize the project:
    ```sh
    go mod tidy
    ```

1. Air reloads the page whenever you save a server source file, i.e. a `.go` file:
    ```sh
    go install github.com/air-verse/air@latest
    ```

1. Template is a Go template engine:
    ```sh
    go install github.com/a-h/templ/cmd/templ@latest
    go get github.com/a-h/templ@latest
    ```

1. Tailwind CSS is a utility-first CSS framework:
    ```sh
    curl -sLO https://github.com/tailwindlabs/tailwindcss/releases/latest/download/tailwindcss-linux-x64
    chmod +x tailwindcss-linux-x64
    sudo mv tailwindcss-linux-x64 tailwindcss
    ```

1. Run the `Watch all` task by pressing `Ctrl+Shift+P` and typing `Tasks: Run Task` and selecting `Watch all`. You will see three tasks in their own terminal windows down the button-right of the screen. Feel free to split the terminal window into three panes and run each task in its own pane.

1. Press F5 to attach the debugger to the server, but whenever Air reloads the page, the debugger will be detached. You can reattach the debugger by pressing F5 again.