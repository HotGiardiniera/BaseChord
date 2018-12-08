#!/usr/bin/env python3
import sys
import argparse
from math import pi

TOP_NODE = "--"
SIDE_NODE = "|"
LEFT_NODE = "/"
RIGHT_NODE = '\\'

def every_other(rng):
    # Takes a range tuple (start,finish) and
    # prints the range with a `-` as every other 
    pass

def draw_ring(nodes, scale):
    # Circle drawing code a modified version of "PM 2Ring" answer at:
    # https://stackoverflow.com/questions/33171682/printing-text-in-form-of-circle
    xscale = scale

    #Maximum diameter, plus a little padding
    width = 3 + int(0.5 + xscale * nodes)

    rad2 = nodes ** 2
    num = 0
    for y in range(-nodes, nodes + 1):
        print_str = "{}".format(num)
        #Find width at this height
        x = int(0.5 + xscale * (rad2 - y ** 2) ** 0.5)
        if y > nodes-2 or y < -(nodes) + 2:
            s = "{}".format(print_str*x)
        else:
            s = "{}{}{}".format(print_str," "*(x-3), print_str)

        
        print(s.center(width))
        if y >=0:
            num -= 1
        else:
            num += 1

def main():
    MAX_SCALE = 17
    MIN_SCALE = 7

    MAX_XSCALE = 5.2
    MIN_XSCALE = 4.2 
    draw_ring(MAX_SCALE, MAX_XSCALE)





if __name__ == "__main__":

    main()
    sys.exit(0)