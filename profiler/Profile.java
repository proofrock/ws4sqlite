import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.file.Files;
import java.nio.file.Paths;
import java.util.concurrent.CountDownLatch;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;
import java.util.stream.Collectors;

public class Profile {
    private static final int NUM_THREADS = 8;

    private static byte[] JSON_BYTES;
    private static int JSON_LEN;

    static {
        try {
            var JSON = Files.readAllLines(Paths.get("./request.json")).stream()
                    .collect(Collectors.joining(""));
            JSON_BYTES = JSON.getBytes("utf-8");
            JSON_LEN = JSON_BYTES.length;
        } catch (Exception e) {
            e.printStackTrace();
            System.exit(1);
        }
    }

    private static int numRequests;
    private static String urlToCall;

    private static ExecutorService threadPool = Executors.newFixedThreadPool(NUM_THREADS);

    public static void main(String[] args) throws Exception {
        numRequests = Integer.parseInt(args[0]);
        urlToCall = args[1];

        var cdl = new CountDownLatch(numRequests);

        var start = System.currentTimeMillis();

        for (int i = 0; i < numRequests; i++) {
            threadPool.execute(() -> {
                performHttpRequest();
                cdl.countDown();
            });
        }
        cdl.await();

        System.out.println((System.currentTimeMillis() - start) / 1000.0);

        threadPool.shutdown();
    }

    private static void performHttpRequest() {
        try {
            var url = new URL(urlToCall);
            var connection = (HttpURLConnection) url.openConnection();
            connection.setRequestMethod("POST");
            connection.setRequestProperty("Content-Type", "application/json"); // Set Content-Type header
            connection.setDoOutput(true);

            try (var os = connection.getOutputStream()) {
                os.write(JSON_BYTES, 0, JSON_LEN);
            }

            var ret = connection.getResponseCode();
            if (ret != 200) {
                try (InputStream errorStream = connection.getErrorStream()) {
                    byte[] buffer = new byte[1024];
                    int bytesRead;
                    while ((bytesRead = errorStream.read(buffer)) != -1) {
                        System.err.write(buffer, 0, bytesRead);
                    }
                }
                try (InputStream errorStream = connection.getInputStream()) {
                    byte[] buffer = new byte[1024];
                    int bytesRead;
                    while ((bytesRead = errorStream.read(buffer)) != -1) {
                        System.err.write(buffer, 0, bytesRead);
                    }
                }
            }
            connection.disconnect();
        } catch (Exception e) {
            e.printStackTrace();
        }
    }
}
