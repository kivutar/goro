package org.goro;

import android.app.Activity;
import android.app.AlertDialog;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.text.InputType;
import android.view.Gravity;
import android.view.KeyEvent;
import android.view.KeyCharacterMap;
import android.view.MotionEvent;
import android.view.Surface;
import android.view.SurfaceHolder;
import android.view.SurfaceView;
import android.view.View;
import android.view.WindowManager;
import android.view.inputmethod.BaseInputConnection;
import android.view.inputmethod.EditorInfo;
import android.view.inputmethod.InputConnection;
import android.view.inputmethod.InputMethodManager;
import android.widget.Button;
import android.widget.FrameLayout;
import android.widget.TextView;
import java.io.File;

public final class GoroActivity extends Activity {
    static { System.loadLibrary("goro"); }
    private static native void nativeStart(Surface surface, int width, int height, String dataDir);
    private static native void nativeStop();
    private static native String nativeStatus();
    private static native void nativePointer(int kind, int button, int buttons, float x, float y);
    private static native void nativeScroll(float x, float y, float delta);
    private static native void nativeKey(int code, int mods, boolean down);
    private static native void nativeText(int codepoint);

    private final Handler handler = new Handler(Looper.getMainLooper());
    private GameView game;
    private File dataDir;
    private boolean running;
    private boolean resumed;
    private final Runnable controllerTick = new Runnable() {
        @Override public void run() {
            if (!running) return;
            if (Math.abs(game.stickX) > 0.15f || Math.abs(game.stickY) > 0.15f) {
                game.pointerX = Math.max(0, Math.min(game.renderWidth - 1, game.pointerX + game.stickX * 12));
                game.pointerY = Math.max(0, Math.min(game.renderHeight - 1, game.pointerY + game.stickY * 12));
                nativePointer(2, -1, game.gamepadButtons, game.pointerX, game.pointerY);
            }
            handler.postDelayed(this, 16);
        }
    };
    private final Runnable statusCheck = new Runnable() {
        @Override public void run() {
            if (!running) return;
            String error = nativeStatus();
            if (!error.isEmpty()) {
                stopGame();
                if (error.equals("@closed")) { finish(); return; }
                new AlertDialog.Builder(GoroActivity.this).setTitle("Goro could not start")
                    .setMessage(error).setPositiveButton("Close", (dialog, which) -> finish()).show();
                return;
            }
            handler.postDelayed(this, 1000);
        }
    };

    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        dataDir = getExternalFilesDir(null);
        if (dataDir == null) dataDir = getFilesDir();
        dataDir.mkdirs();
        FrameLayout root = new FrameLayout(this);
        game = new GameView();
        root.addView(game, new FrameLayout.LayoutParams(-1, -1));
        if (!new File(dataDir, "data.grf").isFile() && !new File(dataDir, "data").isDirectory()) {
            TextView instructions = new TextView(this);
            instructions.setText("Copy your Ragnarok client data to:\n" + dataDir + "\nThen reopen Goro.");
            instructions.setTextSize(20);
            instructions.setGravity(Gravity.CENTER);
            root.addView(instructions, new FrameLayout.LayoutParams(-1, -1));
        }
        Button keyboard = new Button(this);
        keyboard.setText("Keyboard");
        keyboard.setAlpha(0.65f);
        keyboard.setOnClickListener(v -> showKeyboard());
        FrameLayout.LayoutParams buttonLayout = new FrameLayout.LayoutParams(-2, -2, Gravity.TOP | Gravity.RIGHT);
        root.addView(keyboard, buttonLayout);
        setContentView(root);
        immersive();
    }

    private void immersive() {
        getWindow().getDecorView().setSystemUiVisibility(View.SYSTEM_UI_FLAG_FULLSCREEN
            | View.SYSTEM_UI_FLAG_HIDE_NAVIGATION | View.SYSTEM_UI_FLAG_IMMERSIVE_STICKY
            | View.SYSTEM_UI_FLAG_LAYOUT_FULLSCREEN | View.SYSTEM_UI_FLAG_LAYOUT_HIDE_NAVIGATION
            | View.SYSTEM_UI_FLAG_LAYOUT_STABLE);
    }

    private void showKeyboard() {
        game.requestFocus();
        ((InputMethodManager)getSystemService(INPUT_METHOD_SERVICE)).showSoftInput(game, InputMethodManager.SHOW_IMPLICIT);
    }

    @Override public void onWindowFocusChanged(boolean focused) {
        super.onWindowFocusChanged(focused);
        if (focused) immersive();
    }
    @Override protected void onResume() {
        super.onResume();
        resumed = true;
        if (game != null) startGame();
    }
    @Override protected void onPause() {
        resumed = false;
        stopGame();
        super.onPause();
    }
    private void startGame() {
        if (!resumed || running || !game.surfaceReady) return;
        if (!new File(dataDir, "data.grf").isFile() && !new File(dataDir, "data").isDirectory()) return;
        nativeStart(game.getHolder().getSurface(), game.renderWidth, game.renderHeight, dataDir.getAbsolutePath());
        running = true;
        handler.postDelayed(statusCheck, 1000);
        handler.post(controllerTick);
    }
    private void stopGame() {
        handler.removeCallbacks(statusCheck);
        handler.removeCallbacks(controllerTick);
        if (!running) return;
        nativeStop();
        running = false;
        game.stickX = game.stickY = 0;
        game.gamepadButtons = 0;
    }

    private static void sendText(CharSequence text) {
        for (int i = 0; i < text.length();) {
            int cp = Character.codePointAt(text, i);
            nativeText(cp);
            i += Character.charCount(cp);
        }
    }
    private static void tapKey(int key) { nativeKey(key, 0, true); nativeKey(key, 0, false); }

    @Override public boolean dispatchKeyEvent(KeyEvent event) {
        int key = event.getKeyCode();
        if (key == KeyEvent.KEYCODE_VOLUME_UP || key == KeyEvent.KEYCODE_VOLUME_DOWN || key == KeyEvent.KEYCODE_POWER) {
            return super.dispatchKeyEvent(event);
        }
        if (!running) return super.dispatchKeyEvent(event);
        if (key == KeyEvent.KEYCODE_BUTTON_SELECT) {
            if (event.getAction() == KeyEvent.ACTION_DOWN && event.getRepeatCount() == 0) showKeyboard();
            return true;
        }
        // A/B operate the pointer; Start opens the game's Escape menu.
        if (key == KeyEvent.KEYCODE_BUTTON_A || key == KeyEvent.KEYCODE_BUTTON_B) {
            int button = key == KeyEvent.KEYCODE_BUTTON_A ? 0 : 2;
            boolean down = event.getAction() == KeyEvent.ACTION_DOWN;
            if (down && event.getRepeatCount() > 0) return true;
            int mask = button == 0 ? 1 : 2;
            if (down) game.gamepadButtons |= mask;
            else game.gamepadButtons &= ~mask;
            nativePointer(down ? 0 : 1, button, game.gamepadButtons, game.pointerX, game.pointerY);
            return true;
        }
        if (key == KeyEvent.KEYCODE_BUTTON_START) key = KeyEvent.KEYCODE_ESCAPE;
        int mods = (event.isShiftPressed() ? 1 : 0) | (event.isCtrlPressed() ? 2 : 0)
            | (event.isAltPressed() ? 4 : 0) | (event.isMetaPressed() ? 8 : 0);
        if (event.getAction() == KeyEvent.ACTION_MULTIPLE && event.getCharacters() != null) {
            sendText(event.getCharacters());
        } else {
            boolean down = event.getAction() == KeyEvent.ACTION_DOWN;
            nativeKey(key, mods, down);
            if (down && !event.isCtrlPressed() && !event.isAltPressed()) {
                int cp = event.getUnicodeChar();
                if (cp >= 32 && (cp & KeyCharacterMap.COMBINING_ACCENT) == 0) nativeText(cp);
            }
        }
        return true;
    }

    private final class GameView extends SurfaceView implements SurfaceHolder.Callback {
        boolean surfaceReady;
        int renderWidth = 1280, renderHeight = 720;
        float pointerX = 640, pointerY = 360;
        float stickX, stickY;
        int gamepadButtons;
        boolean rightTouch;
        boolean touchReleased;
        float pinchDistance;

        GameView() {
            super(GoroActivity.this);
            setFocusable(true);
            setFocusableInTouchMode(true);
            getHolder().addCallback(this);
            getHolder().setFixedSize(renderWidth, renderHeight);
            requestFocus();
        }
        @Override public void surfaceCreated(SurfaceHolder holder) {}
        @Override public void surfaceChanged(SurfaceHolder holder, int format, int width, int height) {
            if (running && (width != renderWidth || height != renderHeight)) stopGame();
            renderWidth = width;
            renderHeight = height;
            surfaceReady = true;
            startGame();
        }
        @Override public void surfaceDestroyed(SurfaceHolder holder) {
            surfaceReady = false;
            stopGame();
        }
        @Override public boolean onCheckIsTextEditor() { return true; }
        @Override public InputConnection onCreateInputConnection(EditorInfo info) {
            info.inputType = InputType.TYPE_CLASS_TEXT | InputType.TYPE_TEXT_VARIATION_VISIBLE_PASSWORD;
            info.imeOptions = EditorInfo.IME_FLAG_NO_EXTRACT_UI | EditorInfo.IME_ACTION_DONE;
            // The fallback connection owns composition and clears its buffer
            // when it forwards committed text through sendKeyEvent below.
            return new BaseInputConnection(this, false) {
                @Override public boolean deleteSurroundingText(int before, int after) {
                    for (int i = 0; i < before; i++) tapKey(KeyEvent.KEYCODE_DEL);
                    for (int i = 0; i < after; i++) tapKey(KeyEvent.KEYCODE_FORWARD_DEL);
                    return true;
                }
                @Override public boolean sendKeyEvent(KeyEvent event) { return dispatchKeyEvent(event); }
                @Override public boolean performEditorAction(int action) {
                    finishComposingText();
                    tapKey(KeyEvent.KEYCODE_ENTER);
                    ((InputMethodManager)getSystemService(INPUT_METHOD_SERVICE)).hideSoftInputFromWindow(getWindowToken(), 0);
                    return true;
                }
            };
        }
        @Override public boolean onTouchEvent(MotionEvent event) {
            if (!running) return true;
            requestFocus();
            pointerX = event.getX() * renderWidth / getWidth();
            pointerY = event.getY() * renderHeight / getHeight();
            int action = event.getActionMasked();
            if (action == MotionEvent.ACTION_DOWN) {
                rightTouch = false;
                touchReleased = false;
                nativePointer(0, 0, 1, pointerX, pointerY);
            } else if (action == MotionEvent.ACTION_POINTER_DOWN && event.getPointerCount() == 2) {
                nativePointer(1, 0, 0, pointerX, pointerY);
                rightTouch = true;
                pinchDistance = distance(event);
                nativePointer(0, 2, 2, pointerX, pointerY);
            } else if (action == MotionEvent.ACTION_MOVE) {
                nativePointer(2, -1, touchReleased ? 0 : (rightTouch ? 2 : 1), pointerX, pointerY);
                if (rightTouch && event.getPointerCount() >= 2) {
                    float distance = distance(event);
                    if (Math.abs(distance - pinchDistance) > 12) {
                        nativeScroll(pointerX, pointerY, (pinchDistance - distance) / 60);
                        pinchDistance = distance;
                    }
                }
            } else if (action == MotionEvent.ACTION_UP || action == MotionEvent.ACTION_CANCEL) {
                if (!touchReleased) nativePointer(1, rightTouch ? 2 : 0, 0, pointerX, pointerY);
                rightTouch = false;
            } else if (action == MotionEvent.ACTION_POINTER_UP && rightTouch) {
                nativePointer(1, 2, 0, pointerX, pointerY);
                touchReleased = true;
            }
            return true;
        }
        private float distance(MotionEvent event) {
            return (float)Math.hypot(event.getX(0) - event.getX(1), event.getY(0) - event.getY(1));
        }
        @Override public boolean onGenericMotionEvent(MotionEvent event) {
            if (!running) return true;
            if (event.isFromSource(android.view.InputDevice.SOURCE_JOYSTICK)) {
                stickX = event.getAxisValue(MotionEvent.AXIS_X);
                stickY = event.getAxisValue(MotionEvent.AXIS_Y);
                if (Math.abs(stickX) < 0.15f) stickX = 0;
                if (Math.abs(stickY) < 0.15f) stickY = 0;
                return true;
            } else {
                pointerX = event.getX() * renderWidth / getWidth();
                pointerY = event.getY() * renderHeight / getHeight();
            }
            nativePointer(2, -1, event.getButtonState(), pointerX, pointerY);
            if (event.getActionMasked() == MotionEvent.ACTION_SCROLL) {
                nativeScroll(pointerX, pointerY, -event.getAxisValue(MotionEvent.AXIS_VSCROLL));
            }
            return true;
        }
    }
}
