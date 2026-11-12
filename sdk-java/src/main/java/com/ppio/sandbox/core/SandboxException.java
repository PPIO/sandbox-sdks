package com.ppio.sandbox.core;

/**
 * Base exception for all PPIO Sandbox errors.
 */
public class SandboxException extends Exception {
    public SandboxException(String message) {
        super(message);
    }

    public SandboxException(String message, Throwable cause) {
        super(message, cause);
    }
}
