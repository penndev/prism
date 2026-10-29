#include <resolv.h>
#include <arpa/inet.h>
#include <string.h>

static char prism_dns_buf[INET_ADDRSTRLEN];

static int keep_dns(const char *ip) {
    return ip[0] != 0 && strcmp(ip, "0.0.0.0") != 0 && strcmp(ip, "127.0.0.1") != 0;
}

const char *prism_system_dns_ipv4(void) {
    struct __res_state state;
    union res_sockaddr_union servers[8];
    int n, i;

    memset(&state, 0, sizeof(state));
    prism_dns_buf[0] = 0;
    if (res_ninit(&state) != 0) return prism_dns_buf;

    n = res_getservers(&state, servers, 8);
    for (i = 0; i < n; i++) {
        if (servers[i].sin.sin_family != AF_INET) continue;
        if (inet_ntop(AF_INET, &servers[i].sin.sin_addr, prism_dns_buf, sizeof(prism_dns_buf)) == NULL) {
            prism_dns_buf[0] = 0;
            continue;
        }
        if (keep_dns(prism_dns_buf)) {
            res_ndestroy(&state);
            return prism_dns_buf;
        }
        prism_dns_buf[0] = 0;
    }

    res_ndestroy(&state);
    return prism_dns_buf;
}
