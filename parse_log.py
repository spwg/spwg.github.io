import sys, glob, re, datetime, json

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("Missing directory path")
        exit(1)
    lines = []
    for file in glob.glob(sys.argv[1] + '/gin*.log'):
        with open(file, "r") as f:
            lines.extend(f.readlines())
    lines = [[x.strip() for x in line.split('|')] for line in lines if len(line) > 5 and line[:5] == "[GIN]"]
    for line in lines:
        time = datetime.datetime.strptime(line[0], "[GIN] %Y/%m/%d - %H:%M:%S")
        time = time.astimezone(datetime.timezone(datetime.timedelta(0)))
        status = int(line[1])
        if line[2][-2:] == "µs":
            latency_ns = float(line[2][:-2])
        elif line[2][-2:] == "ms":
            latency_ns = float(line[2][:-2]) * 1000
        else:
            print("unrecognized time format", line[2])
            exit(1)
        ip = line[3]
        spl = line[4].split()
        method = spl[0]
        route = spl[1]
        # print(json.dumps({
        #     "time": str(time), "status": status, "latency_ns": latency_ns, "ip": ip, "method": method, "route": route,
        # }))
        fmt_str = "%d/%b/%Y:%H:%M:%S %z"
        print(ip, "-", "-", "[" + time.strftime(fmt_str) + "]", "\"" + method + " " + route + "\"", status, "-")

