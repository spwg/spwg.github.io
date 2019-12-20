This website exists because I wanted to buy my name's domain name. Just hacking around on it.
The CNAME file tells Github to redirect traffic from spwg.github.io to spencerwgreene.com.

The code that runs the website is in the http directory. Main.go runs a gin-gonic web server
written in Go that serves static files from the http/site directory. To run the web server
in development, clone the repository, open a command line prompt, and run
```console
$ git clone https://github.com/spwg/spwg.github.io.git
$ cd spwg.github.io/http
$ go run main.go &
$ tail -f server.log
```
The web server runs on port 8080 by default. So, go to localhost:8080 in a browser to see
the web site running.

To deploy the website, ssh into the GCP instance, run tmux, clone it if it's not already cloned, 
change directory into the git repository, and run
```console
$ sudo iptables -A PREROUTING -t nat -p tcp --dport 80 -j REDIRECT --to-ports 8080
$ export GIN_MODE=release
$ go run main.go &
```
The first line redirects incoming connections for port 80 to port 8080, where the website
is running. The -A flag makes the redirection append to the PREROUTING chain, which alters
incoming packets as soon as they arrive, i.e. before routing them. The '-t nat' flag alters
the nat table, network address translation. The '-p tcp' flag makes the alteration just for
tcp packets, not udp, icmp, or anything else. The '--dport 80' flag says the destination of
the incoming packets that match this rule is port 80. The '-j REDIRECT --to-ports 8080' flag
makes the incoming request redirect to port 8080.
