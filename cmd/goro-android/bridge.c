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
static JavaVM *document_vm;
static jclass document_class;
static jmethodID document_list;
static jmethodID document_open;

// All resource callbacks run in one C call on the current Go thread. Attach
// only for its duration; no Activity or per-game references are retained.
static int document_env(JNIEnv **env, int *attached) {
    if (!document_vm) return 0;
    jint status = (*document_vm)->GetEnv(document_vm, (void **)env, JNI_VERSION_1_6);
    *attached = (status == JNI_EDETACHED);
    if (*attached) {
        if ((*document_vm)->AttachCurrentThread(document_vm, env, NULL) != JNI_OK) return 0;
    } else if (status != JNI_OK) {
        return 0;
    }
    if ((**env)->PushLocalFrame(*env, 4) != JNI_OK) {
        if ((**env)->ExceptionCheck(*env)) (**env)->ExceptionClear(*env);
        if (*attached) (*document_vm)->DetachCurrentThread(document_vm);
        return 0;
    }
    return 1;
}

static void document_done(JNIEnv *env, int attached) {
    if ((*env)->ExceptionCheck(env)) (*env)->ExceptionClear(env);
    (*env)->PopLocalFrame(env, NULL);
    if (attached) (*document_vm)->DetachCurrentThread(document_vm);
}

char *GoroDocumentList(const char *uri) {
    JNIEnv *env = NULL;
    int attached = 0;
    if (!document_env(&env, &attached)) return NULL;
    jstring location = (*env)->NewStringUTF(env, uri);
    jbyteArray bytes = location ? (jbyteArray)(*env)->CallStaticObjectMethod(env, document_class, document_list, location) : NULL;
    char *result = NULL;
    if (!(*env)->ExceptionCheck(env) && bytes) {
        jsize size = (*env)->GetArrayLength(env, bytes);
        result = malloc((size_t)size + 1);
        if (result) {
            (*env)->GetByteArrayRegion(env, bytes, 0, size, (jbyte *)result);
            result[size] = 0;
        }
    }
    document_done(env, attached);
    return result;
}

int GoroDocumentOpen(const char *uri) {
    JNIEnv *env = NULL;
    int attached = 0;
    if (!document_env(&env, &attached)) return -1;
    jstring location = (*env)->NewStringUTF(env, uri);
    int fd = location ? (*env)->CallStaticIntMethod(env, document_class, document_open, location) : -1;
    if ((*env)->ExceptionCheck(env)) fd = -1;
    document_done(env, attached);
    return fd;
}

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

JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeStart(JNIEnv *env, jclass cls, jobject surface, jint width, jint height, jstring path, jstring source) {
    (void)cls;
    pthread_once(&log_once, redirect_logs);
    if (!document_class) {
        jclass type = (*env)->FindClass(env, "org/goro/DocumentTree");
        if (!type) return;
        document_class = (*env)->NewGlobalRef(env, type);
        (*env)->DeleteLocalRef(env, type);
        if (!document_class) return;
        document_list = (*env)->GetStaticMethodID(env, document_class, "list", "(Ljava/lang/String;)[B");
        document_open = (*env)->GetStaticMethodID(env, document_class, "open", "(Ljava/lang/String;)I");
        if (!document_list || !document_open) return;
        (*env)->GetJavaVM(env, &document_vm);
    }
    if (window) return;
    window = ANativeWindow_fromSurface(env, surface);
    if (!window) return;
    const char *directory = (*env)->GetStringUTFChars(env, path, NULL);
    if (!directory) { ANativeWindow_release(window); window = NULL; return; }
    const char *data_source = (*env)->GetStringUTFChars(env, source, NULL);
    if (!data_source) {
        (*env)->ReleaseStringUTFChars(env, path, directory);
        ANativeWindow_release(window); window = NULL; return;
    }
    GoroStart((uintptr_t)window, width, height, (char *)directory, (char *)data_source);
    (*env)->ReleaseStringUTFChars(env, source, data_source);
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

JNIEXPORT jboolean JNICALL Java_org_goro_GoroActivity_nativeCanChooseFolder(JNIEnv *env, jclass cls) {
    (void)env; (void)cls;
    return GoroCanChooseFolder() ? JNI_TRUE : JNI_FALSE;
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

JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeFocus(JNIEnv *env, jclass cls, jboolean focused) {
    (void)env; (void)cls;
    GoroFocus(focused);
}

JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeGamepadDevice(JNIEnv *env, jclass cls, jint id, jstring name, jboolean connected) {
    (void)cls;
    const char *text = (*env)->GetStringUTFChars(env, name, NULL);
    if (!text) return;
    GoroGamepadDevice(id, (char *)text, connected);
    (*env)->ReleaseStringUTFChars(env, name, text);
}
JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeGamepadKey(JNIEnv *env, jclass cls, jint id, jint key, jboolean down) {
    (void)env; (void)cls;
    GoroGamepadKey(id, key, down);
}
JNIEXPORT void JNICALL Java_org_goro_GoroActivity_nativeGamepadMotion(JNIEnv *env, jclass cls, jint id, jfloat lx, jfloat ly, jfloat rx, jfloat ry, jfloat lt, jfloat rt, jfloat hx, jfloat hy) {
    (void)env; (void)cls;
    GoroGamepadMotion(id, lx, ly, rx, ry, lt, rt, hx, hy);
}
