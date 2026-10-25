#include <resolv.h>
#include <arpa/inet.h>
#include <string.h>

static char prism_dns_buf[INET_ADDRSTRLEN];

const char *prism_system_dns_ipv4(void) {
    struct __res_state state;
    memset(&state, 0, sizeof(state));
    prism_dns_buf[0] = 0;
    if (res_ninit(&state) != 0) return prism_dns_buf;
    int n = state.nscount;
    if (n > 3) n = 3;
    for (int i = 0; i < n; i++) {
        if (state.nsaddr_list[i].sin_family != AF_INET) continue;
        if (inet_ntop(AF_INET, &state.nsaddr_list[i].sin_addr, prism_dns_buf, sizeof(prism_dns_buf)) == NULL) {
            prism_dns_buf[0] = 0;
            continue;
        }
        if (strcmp(prism_dns_buf, "0.0.0.0") == 0 || strcmp(prism_dns_buf, "127.0.0.1") == 0) {
            prism_dns_buf[0] = 0;
            continue;
        }
        break;
    }
    res_ndestroy(&state);
    return prism_dns_buf;
}
