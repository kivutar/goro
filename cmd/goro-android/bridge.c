//go:build android && cgo

#include <jni.h>
#include <android/native_window_jni.h>
#include <android/log.h>
#include <pthread.h>
#include <unistd.h>
#include <stdlib.h>
#include <stdint.h>
#include "_cgo_export.h"

static ANativeWindow *window;
static pthread_once_t log_once = PTHREAD_ONCE_INIT;

static void *read_logs(void *arg) {
    int fd = (int)(intptr_t)arg;
    char line[4096];
    size_t used = 0;
    char ch;
    while (read(fd, &ch, 1) == 1) {
        if (ch == '\n' || used == sizeof(line) - 1) {
            line[used] = 0;
            __android_log_write(ANDROID_LOG_INFO, "Goro", line);
            used = 0;
        }
        if (ch != '\n') line[used++] = ch;
    }
    close(fd);
    return NULL;
}

static void redirect_logs(void) {
    int fds[2];
    if (pipe(fds) != 0) return;
    pthread_t thread;
    if (pthread_create(&thread, NULL, read_logs, (void *)(intptr_t)fds[0]) != 0) {
        close(fds[0]); close(fds[1]); return;
    }
    pthread_detach(thread);
    dup2(fds[1], STDOUT_FILENO);
    dup2(fds[1], STDERR_FILENO);
    close(fds[1]);
}

JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeStart(JNIEnv *env, jclass cls, jobject surface, jint width, jint height, jstring path) {
    (void)cls;
    pthread_once(&log_once, redirect_logs);
    if (window) return;
    window = ANativeWindow_fromSurface(env, surface);
    if (!window) return;
    const char *directory = (*env)->GetStringUTFChars(env, path, NULL);
    if (!directory) { ANativeWindow_release(window); window = NULL; return; }
    GoroStart((uintptr_t)window, width, height, (char *)directory);
    (*env)->ReleaseStringUTFChars(env, path, directory);
}

JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeStop(JNIEnv *env, jclass cls) {
    (void)env; (void)cls;
    GoroStop();
    if (window) { ANativeWindow_release(window); window = NULL; }
}

JNIEXPORT jstring JNICALL Java_org_goro_GoroActivity_nativeStatus(JNIEnv *env, jclass cls) {
    (void)cls;
    char *status = GoroStatus();
    jstring result = (*env)->NewStringUTF(env, status);
    free(status);
    return result;
}

JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativePointer(JNIEnv *env, jclass cls, jint kind, jint button, jint buttons, jfloat x, jfloat y) {
    (void)env; (void)cls;
    GoroPointer(kind, button, buttons, x, y);
}
JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeScroll(JNIEnv *env, jclass cls, jfloat x, jfloat y, jfloat delta) {
    (void)env; (void)cls;
    GoroScroll(x, y, delta);
}
JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeKey(JNIEnv *env, jclass cls, jint code, jint mods, jboolean down) {
    (void)env; (void)cls;
    GoroKey(code, mods, down);
}
JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeText(JNIEnv *env, jclass cls, jint codepoint) {
    (void)env; (void)cls;
    GoroText(codepoint);
}
