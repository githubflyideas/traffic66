package main

// The release build links Linux binaries statically and routes getentropy
// (used by the C++ runtime's random_device) here with -Wl,--wrap, so the
// program also runs on kernels without the getrandom system call, such as
// CentOS 7's 3.10.

/*
#include <errno.h>
#include <fcntl.h>
#include <stddef.h>
#include <unistd.h>
#include <sys/syscall.h>

int __wrap_getentropy(void *buf, size_t len) {
	if (len > 256) { errno = EIO; return -1; }
#ifdef SYS_getrandom
	long r = syscall(SYS_getrandom, buf, len, 0);
	if (r == (long)len) return 0;
#endif
	int fd = open("/dev/urandom", O_RDONLY | O_CLOEXEC);
	if (fd < 0) return -1;
	size_t got = 0;
	while (got < len) {
		ssize_t n = read(fd, (char *)buf + got, len - got);
		if (n < 0 && errno == EINTR) continue;
		if (n <= 0) { close(fd); errno = EIO; return -1; }
		got += (size_t)n;
	}
	close(fd);
	return 0;
}
*/
import "C"
