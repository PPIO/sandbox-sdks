package com.ppio.sandbox.core;

import static org.junit.jupiter.api.Assertions.assertEquals;
import org.junit.jupiter.api.Test;

class PpioBrandingTest {
    @Test
    void usesPpioDefaultDomain() {
        assertEquals("sandbox.ppio.cn", Sandbox.DEFAULT_PPIO_DOMAIN);
    }
}
