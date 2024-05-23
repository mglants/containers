#!/usr/bin/env python

def get_latest(channel):
    return "v0.0.1"

if __name__ == "__main__":
    import sys
    channel = sys.argv[1]
    print(get_latest(channel))