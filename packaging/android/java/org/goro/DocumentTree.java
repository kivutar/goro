package org.goro;

import android.content.ContentResolver;
import android.content.Context;
import android.database.Cursor;
import android.net.Uri;
import android.os.ParcelFileDescriptor;
import android.provider.DocumentsContract;
import java.io.IOException;
import java.nio.charset.StandardCharsets;
import org.json.JSONArray;
import org.json.JSONObject;

// The native resource reader opens provider descriptors directly. No asset copy
// or filesystem-path conversion is needed, including for removable storage.
public final class DocumentTree {
    private static ContentResolver resolver;

    public static void init(Context context) {
        resolver = context.getApplicationContext().getContentResolver();
    }

    public static byte[] list(String location) throws Exception {
        Uri folder = Uri.parse(location);
        String id = folder.getPathSegments().contains("document")
            ? DocumentsContract.getDocumentId(folder) : DocumentsContract.getTreeDocumentId(folder);
        Uri children = DocumentsContract.buildChildDocumentsUriUsingTree(folder, id);
        String[] columns = {
            DocumentsContract.Document.COLUMN_DOCUMENT_ID,
            DocumentsContract.Document.COLUMN_DISPLAY_NAME,
            DocumentsContract.Document.COLUMN_MIME_TYPE,
            DocumentsContract.Document.COLUMN_SIZE,
            DocumentsContract.Document.COLUMN_LAST_MODIFIED
        };
        JSONArray entries = new JSONArray();
        try (Cursor cursor = resolver.query(children, columns, null, null, null)) {
            if (cursor == null) throw new IOException("Cannot read the selected folder");
            while (cursor.moveToNext()) {
                JSONObject entry = new JSONObject();
                entry.put("name", cursor.getString(1));
                entry.put("uri", DocumentsContract.buildDocumentUriUsingTree(folder, cursor.getString(0)).toString());
                entry.put("directory", DocumentsContract.Document.MIME_TYPE_DIR.equals(cursor.getString(2)));
                entry.put("size", cursor.isNull(3) ? 0 : cursor.getLong(3));
                entry.put("modified", cursor.isNull(4) ? 0 : cursor.getLong(4));
                entries.put(entry);
            }
        }
        return entries.toString().getBytes(StandardCharsets.UTF_8);
    }

    public static int open(String location) throws IOException {
        try (ParcelFileDescriptor file = resolver.openFileDescriptor(Uri.parse(location), "r")) {
            if (file == null) throw new IOException("Cannot open " + location);
            return file.detachFd(); // Ownership transfers to Go's os.File.
        }
    }
}
