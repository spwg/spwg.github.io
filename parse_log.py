import sys, glob, re, datetime

if __name__ == "__main__":
    if len(sys.argv) != 2:
        print("usage: python3 parse_log.py <logs directory>")
        exit(1)
    last_file, ft = None, None
    for file in glob.glob(sys.argv[1] + '/gin*.log'):
        s = ' '.join(file.split()[1:])
        fmt_str = "%d %b %y %H:%M UTC.log"
        t = datetime.datetime.strptime(s, fmt_str)
        t = t.replace(tzinfo=datetime.timezone(datetime.timedelta(0)))
        if last_file is None or t > ft:
            last_file, ft = file, t
    assert last_file is not None
    print("Most recent file is \"%s\"." % last_file)
    lines = []
    with open(last_file, "r") as f:
        lines.extend(f.readlines())
    for line in lines:
        line = line.strip()
        print(line)
        # TODO: parse the log line into a common log format for 

