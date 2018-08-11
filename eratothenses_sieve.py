#!/usr/local/bin/python3
import timeit


def sieve(lim=1000000):
    """Generator for prime numbers up to a limit."""
    primes = [True] * lim
    # start at 2 because 0 and 1 are prime
    for i in range(2, len(primes)):
        if primes[i]:
            for j in range(i + 1, len(primes)):
                if j % i == 0:
                    primes[j] = False
            yield i


if __name__ == '__main__':
    for i in range(1000, 1000000, 1000):
        now = timeit.default_timer()
        list(sieve(lim=i))
        then = timeit.default_timer()
        print(str(i) + ',', then - now)
