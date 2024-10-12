import re

module_header = re.compile(r'-- *module *: *([\w|\.]+) *\n')


class SingleFilePackage:
    def __init__(self, text):
        self.text = text

    def provide_files(self) -> list[tuple[str, str]]:
        file_list = module_header.split(self.text)[1:]
        files = []
        for i in range(0, len(file_list), 2):
            module_name = file_list[i]
            file_content = file_list[i + 1]
            files.append((module_name, file_content))
        return files


def run_package(pkg) -> list[tuple[str, str]]:
    files: list[tuple[str, str]] = pkg.provide_files()
    return files
