package io.temporal.demo.wealthmanagement.common;

import java.util.HashMap;
import java.util.Locale;
import java.util.Map;

public class ServerInfo {

    public static String getCertPath() {
        return getEnv("TEMPORAL_CERT_PATH", "");
    }

    public static String getKeyPath() {
        // check first for a specific pkcs8 env var, if null/empty then check for generic env var
        String keyPath = System.getenv("TEMPORAL_KEY_PKCS8_PATH");
        return keyPath != null && !keyPath.isEmpty() ? keyPath : getEnv("TEMPORAL_KEY_PATH", "");
    }

    public static String getNamespace() {
        return getEnv("TEMPORAL_NAMESPACE", "default");
    }

    public static String getAddress() {
        return getEnv("TEMPORAL_ADDRESS", "localhost:7233");
    }

    public static String getTaskqueue() {
        return getEnv("TEMPORAL_TASK_QUEUE", "Supervisor");
    }

    public static String getTaskqueueOpenAccount() {
        return getEnv("TEMPORAL_TASK_QUEUE_OPEN_ACCOUNT", getTaskqueue());
    }

    // Accepts the same values as the Go and Python components (true, t, yes, y or 1, in any case),
    // so they all agree on whether payloads are claim-checked.
    public static Boolean useClaimCheck() {
        String value = getEnv("USE_CLAIM_CHECK", "false").trim().toLowerCase(Locale.ROOT);
        return value.equals("true") || value.equals("t") || value.equals("yes") || value.equals("y") || value.equals("1");
    }

    // A one-line description of whether the claim check is on, logged at startup whatever its state.
    public static String claimCheckStatus() {
        String value = System.getenv("USE_CLAIM_CHECK");
        String setting = value == null || value.isEmpty() ? "USE_CLAIM_CHECK is not set" : "USE_CLAIM_CHECK=" + value;
        return (useClaimCheck() ? "Claim check: ON (" : "Claim check: OFF (") + setting + ")";
    }

    public static Map<String, String> getServerInfo() {
        Map<String, String> info = new HashMap<>();
        info.put("certPath", getCertPath());
        info.put("keyPath", getKeyPath());
        info.put("namespace", getNamespace());
        info.put("address", getAddress());
        info.put("taskQueue", getTaskqueue());
        info.put("useClaimCheck", useClaimCheck().toString());
        return info;
    }

    private static String getEnv(String key, String defaultValue) {
        String value = System.getenv(key);
        return value != null && !value.isEmpty() ? value : defaultValue;
    }
}
