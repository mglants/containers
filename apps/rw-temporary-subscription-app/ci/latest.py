#!/usr/bin/env python

def get_latest(channel):
    return "v2.0.0"

if __name__ == "__main__":
    import sys
    channel = sys.argv[1]
    print(get_latest(channel))
