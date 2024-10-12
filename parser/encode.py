from curses.ascii import isalpha

symbol_to_name = {
    '+': 'p',
    '-': 'm',
    '*': 't',
    '!': 'e',
    '#': 'h',
    '$': 'd',
    '.': 'o',
    '=': 'q',
    "'": 'a',
    "%": 'c',
    "|": 'b',
    "~": 'r',
    ":": 'i',
    "&": 'f',
    "/": 's',
    "\\": 'u',
    "<": 'l',
    ">": 'g',
    "@": 'n',
    "?": 'k',
    '^': 'j'
}

name_to_symbol = {
    'p': '+',
    'm': '-',
    't': '*',
    'e': '!',
    'h': '#',
    'd': '$',
    'o': '.',
    'q': '=',
    'a': "'",
    'c': '%',
    'b': '|',
    'r': '~',
    'i': ':',
    'f': '&',
    's': '/',
    'u': '\\',
    'l': '<',
    'g': '>',
    'n': '@',
    'k': '?',
    'j': '^'}


def encode(text: str) -> str:

    if isalpha(text[0]) or (text[0] == '_'):
        if text.endswith("'"):
            number_of_primes = len([c for c in text if c == "'"])
            rest = "".join([c for c in text if c != "'"])
            return f"XP{number_of_primes}{rest}"
        else:
            return text

    if any(c in symbol_to_name for c in text):
        encoded = ''.join([symbol_to_name[c] for c in text])
        return f'XO{encoded}'
    else:
        return text


def decode(text: str) -> str:
    if text.startswith('XO'):
        encoded = text[2:]
        decoded = ''.join([name_to_symbol[c] for c in encoded])
        return decoded
    if text.startswith('XP'): # Variables with primes, e.g. let x' = 2 in x'
        number_of_primes = int(text[2])
        return text[3:] + (number_of_primes * "'")
    else:
        return text
