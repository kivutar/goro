package org.goro;

import android.app.Activity;
import android.app.AlertDialog;
import android.content.ActivityNotFoundException;
import android.content.Intent;
import android.hardware.input.InputManager;
import android.net.Uri;
import android.os.Bundle;
import android.os.Handler;
import android.os.Looper;
import android.provider.DocumentsContract;
import android.text.InputType;
import android.view.Gravity;
import android.view.InputDevice;
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

public final class GoroActivity extends Activity implements InputManager.InputDeviceListener {
    static { System.loadLibrary("goro"); }
    private static final int PICK_RO_FOLDER = 1;
    private static native void nativeStart(Surface surface, int width, int height, String appDir, String dataSource);
    private static native void nativeStop();
    private static native String nativeStatus();
    private static native boolean nativeCanChooseFolder();
    private static native void nativePointer(int kind, int button, int buttons, float x, float y);
    private static native void nativeScroll(float x, float y, float delta);
    private static native void nativeKey(int code, int mods, boolean down);
    private static native void nativeText(int codepoint);
    private static native void nativeFocus(boolean focused);
    private static native void nativeGamepadDevice(int id, String name, boolean connected);
    private static native void nativeGamepadKey(int id, int key, boolean down);
    private static native void nativeGamepadMotion(int id, float lx, float ly, float rx, float ry,
        float lt, float rt, float hx, float hy);

    private final Handler handler = new Handler(Looper.getMainLooper());
    private GameView game;
    private File dataDir;
    private String dataSource;
    private TextView instructions;
    private Button folder;
    private boolean choosingFolder;
    private boolean startupFailed;
    private boolean running;
    private boolean resumed;
    private InputManager inputManager;
    private final Runnable statusCheck = new Runnable() {
        @Override public void run() {
            if (!running) return;
            folder.setVisibility(nativeCanChooseFolder() ? View.VISIBLE : View.GONE);
            String error = nativeStatus();
            if (!error.isEmpty()) {
                stopGame();
                if (error.equals("@closed")) { finish(); return; }
                startupFailed = true;
                updateFolderPrompt();
                new AlertDialog.Builder(GoroActivity.this).setTitle("Goro could not start")
                    .setMessage(error)
                    .setPositiveButton("Choose RO folder", (dialog, which) -> chooseFolder())
                    .setNeutralButton("Retry", (dialog, which) -> {
                        startupFailed = false;
                        updateFolderPrompt();
                        startGame();
                    })
                    .setNegativeButton("Close", (dialog, which) -> finish()).show();
                return;
            }
            handler.postDelayed(this, 1000);
        }
    };

    @Override public void onCreate(Bundle state) {
        super.onCreate(state);
        inputManager = (InputManager)getSystemService(INPUT_SERVICE);
        inputManager.registerInputDeviceListener(this, handler);
        getWindow().addFlags(WindowManager.LayoutParams.FLAG_KEEP_SCREEN_ON);
        dataDir = getExternalFilesDir(null);
        if (dataDir == null) dataDir = getFilesDir();
        dataDir.mkdirs();
        DocumentTree.init(this);
        dataSource = getPreferences(MODE_PRIVATE).getString("ro_folder", dataDir.getAbsolutePath());
        choosingFolder = state != null && state.getBoolean("choosing_folder", false);
        FrameLayout root = new FrameLayout(this);
        game = new GameView();
        root.addView(game, new FrameLayout.LayoutParams(-1, -1));
        instructions = new TextView(this);
        instructions.setText("Choose your extracted Ragnarok Online folder\nusing the RO folder button.");
        instructions.setTextSize(20);
        instructions.setGravity(Gravity.CENTER);
        root.addView(instructions, new FrameLayout.LayoutParams(-1, -1));
        updateFolderPrompt();
        folder = new Button(this);
        folder.setText("RO folder");
        folder.setAlpha(0.65f);
        folder.setOnClickListener(v -> chooseFolder());
        root.addView(folder, new FrameLayout.LayoutParams(-2, -2, Gravity.TOP | Gravity.LEFT));
        Button keyboard = new Button(this);
        keyboard.setText("Keyboard");
        keyboard.setAlpha(0.65f);
        keyboard.setOnClickListener(v -> showKeyboard());
        FrameLayout.LayoutParams buttonLayout = new FrameLayout.LayoutParams(-2, -2, Gravity.TOP | Gravity.RIGHT);
        root.addView(keyboard, buttonLayout);
        setContentView(root);
        immersive();
    }

    @Override protected void onSaveInstanceState(Bundle state) {
        state.putBoolean("choosing_folder", choosingFolder);
        super.onSaveInstanceState(state);
    }

    private boolean hasClientData() {
        if (dataSource.startsWith("content://")) return true; // Validated by the resource reader.
        File[] files = new File(dataSource).listFiles();
        if (files == null) return false;
        for (File file : files) {
            String name = file.getName();
            if (file.isDirectory() && name.equalsIgnoreCase("data")) return true;
            if (file.isFile() && (name.equalsIgnoreCase("DATA.INI") || name.equalsIgnoreCase("data.grf")
                || name.equalsIgnoreCase("fdata.grf") || name.equalsIgnoreCase("rdata.grf")
                || name.equalsIgnoreCase("sdata.grf"))) return true;
        }
        return false;
    }

    private void updateFolderPrompt() {
        instructions.setVisibility(startupFailed || !hasClientData() ? View.VISIBLE : View.GONE);
    }

    private void chooseFolder() {
        // The UI is refreshed periodically; reject a stale click after login.
        if (running && !nativeCanChooseFolder()) return;
        choosingFolder = true;
        stopGame();
        Intent intent = new Intent(Intent.ACTION_OPEN_DOCUMENT_TREE);
        intent.addFlags(Intent.FLAG_GRANT_READ_URI_PERMISSION | Intent.FLAG_GRANT_PERSISTABLE_URI_PERMISSION
            | Intent.FLAG_GRANT_PREFIX_URI_PERMISSION);
        intent.putExtra(Intent.EXTRA_LOCAL_ONLY, true);
        if (dataSource.startsWith("content://")) {
            intent.putExtra(DocumentsContract.EXTRA_INITIAL_URI, Uri.parse(dataSource));
        }
        try {
            startActivityForResult(intent, PICK_RO_FOLDER);
        } catch (ActivityNotFoundException error) {
            choosingFolder = false;
            new AlertDialog.Builder(this).setTitle("Folder picker unavailable")
                .setMessage("Enable the Android Files app to choose your RO folder.")
                .setPositiveButton("OK", (dialog, which) -> startGame()).show();
        }
    }

    @Override protected void onActivityResult(int request, int result, Intent data) {
        super.onActivityResult(request, result, data);
        if (request != PICK_RO_FOLDER) return;
        choosingFolder = false;
        if (result == RESULT_OK && data != null && data.getData() != null) {
            Uri uri = data.getData();
            try {
                getContentResolver().takePersistableUriPermission(uri, Intent.FLAG_GRANT_READ_URI_PERMISSION);
                String previous = dataSource;
                dataSource = uri.toString();
                getPreferences(MODE_PRIVATE).edit().putString("ro_folder", dataSource).apply();
                startupFailed = false;
                if (!previous.equals(dataSource) && previous.startsWith("content://")) {
                    try {
                        getContentResolver().releasePersistableUriPermission(Uri.parse(previous), Intent.FLAG_GRANT_READ_URI_PERMISSION);
                    } catch (SecurityException ignored) { }
                }
            } catch (SecurityException error) {
                new AlertDialog.Builder(this).setTitle("Folder access unavailable")
                    .setMessage("Please select the folder again and allow Goro to read it.")
                    .setPositiveButton("Choose RO folder", (dialog, which) -> chooseFolder())
                    .setNegativeButton("Cancel", null).show();
            }
        }
        updateFolderPrompt();
        startGame();
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
        if (running) {
            nativeFocus(focused);
            for (int id : InputDevice.getDeviceIds()) {
                if (focused) onInputDeviceAdded(id);
                else nativeGamepadDevice(id, "", false);
            }
        }
    }
    @Override protected void onDestroy() {
        if (inputManager != null) inputManager.unregisterInputDeviceListener(this);
        super.onDestroy();
    }
    private static boolean isGamepad(InputDevice device) {
        return device != null && (device.supportsSource(InputDevice.SOURCE_GAMEPAD)
            || device.supportsSource(InputDevice.SOURCE_JOYSTICK));
    }
    @Override public void onInputDeviceAdded(int id) {
        InputDevice device = InputDevice.getDevice(id);
        if (running && isGamepad(device)) nativeGamepadDevice(id, device.getName(), true);
    }
    @Override public void onInputDeviceRemoved(int id) {
        if (running) nativeGamepadDevice(id, "", false);
    }
    @Override public void onInputDeviceChanged(int id) {
        onInputDeviceRemoved(id);
        onInputDeviceAdded(id);
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
        if (!resumed || running || choosingFolder || startupFailed || !game.surfaceReady || !hasClientData()) return;
        nativeStart(game.getHolder().getSurface(), game.renderWidth, game.renderHeight, dataDir.getAbsolutePath(), dataSource);
        running = true;
        nativeFocus(hasWindowFocus());
        handler.postDelayed(statusCheck, 1000);
        for (int id : InputDevice.getDeviceIds()) onInputDeviceAdded(id);
    }
    private void stopGame() {
        handler.removeCallbacks(statusCheck);
        if (!running) return;
        nativeStop();
        running = false;
        folder.setVisibility(View.VISIBLE);
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
        if (isGamepad(event.getDevice()) && (KeyEvent.isGamepadButton(key)
            || key == KeyEvent.KEYCODE_DPAD_UP || key == KeyEvent.KEYCODE_DPAD_DOWN
            || key == KeyEvent.KEYCODE_DPAD_LEFT || key == KeyEvent.KEYCODE_DPAD_RIGHT)) {
            onInputDeviceAdded(event.getDeviceId());
            boolean down = event.getAction() == KeyEvent.ACTION_DOWN;
            nativeGamepadKey(event.getDeviceId(), key, down);
            if (key == KeyEvent.KEYCODE_BUTTON_SELECT && down && event.getRepeatCount() == 0) showKeyboard();
            return true;
        }
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
                @Override public boolean sendKeyEvent(KeyEvent event) {
                    // GameView also inherits dispatchKeyEvent; route IME input
                    // through the Activity so it reaches the native game.
                    return GoroActivity.this.dispatchKeyEvent(event);
                }
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
        private float gamepadAxisValue(MotionEvent event, int axis, int sample) {
            return sample < event.getHistorySize()
                ? event.getHistoricalAxisValue(axis, sample) : event.getAxisValue(axis);
        }
        @Override public boolean onGenericMotionEvent(MotionEvent event) {
            if (!running) return true;
            if (event.isFromSource(android.view.InputDevice.SOURCE_JOYSTICK)) {
                onInputDeviceAdded(event.getDeviceId());
                InputDevice device = event.getDevice();
                int rx = MotionEvent.AXIS_Z, ry = MotionEvent.AXIS_RZ;
                if (device != null && device.getMotionRange(rx, event.getSource()) == null) {
                    rx = MotionEvent.AXIS_RX; ry = MotionEvent.AXIS_RY;
                }
                // Replay the batch oldest first so a D-pad tap is not lost when
                // its release is the current sample. Analog state ends at the latest value.
                for (int sample = 0; sample <= event.getHistorySize(); sample++) {
                    nativeGamepadMotion(event.getDeviceId(), gamepadAxisValue(event, MotionEvent.AXIS_X, sample),
                        gamepadAxisValue(event, MotionEvent.AXIS_Y, sample), gamepadAxisValue(event, rx, sample), gamepadAxisValue(event, ry, sample),
                        Math.max(gamepadAxisValue(event, MotionEvent.AXIS_LTRIGGER, sample), gamepadAxisValue(event, MotionEvent.AXIS_BRAKE, sample)),
                        Math.max(gamepadAxisValue(event, MotionEvent.AXIS_RTRIGGER, sample), gamepadAxisValue(event, MotionEvent.AXIS_GAS, sample)),
                        gamepadAxisValue(event, MotionEvent.AXIS_HAT_X, sample), gamepadAxisValue(event, MotionEvent.AXIS_HAT_Y, sample));
                }
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
