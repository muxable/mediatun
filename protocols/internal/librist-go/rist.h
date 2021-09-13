#ifndef RIST_H
#define RIST_H

#include <librist/librist.h>

extern int cb_auth_connect(void *, int);
extern int cb_auth_disconnect(void *, int, int);
extern void cb_connection_status(void *, struct rist_peer *, enum rist_connection_status);
extern int cb_recv_oob(void *, const struct rist_oob_block *);
extern int cb_stats(void *, const struct rist_stats *);
extern int cb_recv(void *arg, struct rist_data_block *b);

#endif