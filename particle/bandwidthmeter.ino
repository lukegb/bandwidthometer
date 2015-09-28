#include "application.h"
//#include "spark_disable_wlan.h" (for faster local debugging only)
#include "neopixel/neopixel.h"

#include <inttypes.h>

#define PIXEL_COUNT 150
#define PIXEL_PIN D2
#define PIXEL_TYPE WS2812B

#define UDP_COMMS_PORT 31814
#define INT_SIZE (sizeof(int32_t))
#define UDP_EXPECTED_SIZE (INT_SIZE * 2)

#define FMT_BUF_SIZE 80

#define RX_BANDWIDTH_MAX 21120000 // bytes per second
#define TX_BANDWIDTH_MAX 1575000 // bytes per second

#define RX_COLOUR strip.Color(0, 150, 0)
#define TX_COLOUR strip.Color(150, 0, 0)
#define RXTX_COLOUR strip.Color(150, 150, 0)
#define NONE_COLOUR strip.Color(0, 0, 0)

Adafruit_NeoPixel strip = Adafruit_NeoPixel(PIXEL_COUNT, PIXEL_PIN, PIXEL_TYPE);

union packetBuffer {
    char raw[UDP_EXPECTED_SIZE];
    struct values {
        int32_t rx;
        int32_t tx;
    } values;
};

UDP Udp;
union packetBuffer packetBuffer;

void setup() {
    strip.begin();
    
    Udp.begin(UDP_COMMS_PORT);
    
    Serial.begin(9600);
    Serial.println("Booted!");
    Serial.print("My IP is: ");
    Serial.println(WiFi.localIP());
    Serial.print("My port is: ");
    Serial.println(UDP_COMMS_PORT);
    
    Serial.println("...here we go!");
}

char fmtBuf[FMT_BUF_SIZE];

void loop() {
    int packetSize = Udp.parsePacket();
    if (packetSize > 0 && packetSize == UDP_EXPECTED_SIZE) {
        Udp.read((char*)&packetBuffer, UDP_EXPECTED_SIZE);
        
        Serial.println("YES!");
        
        snprintf(fmtBuf, FMT_BUF_SIZE, "RX: %ld", packetBuffer.values.rx);
        Serial.println(fmtBuf);
        snprintf(fmtBuf, FMT_BUF_SIZE, "TX: %ld", packetBuffer.values.tx);
        Serial.println(fmtBuf);
        
        int rxLedsLit = (packetBuffer.values.rx * PIXEL_COUNT) / RX_BANDWIDTH_MAX;
        int txLedsLit = (packetBuffer.values.tx * PIXEL_COUNT) / TX_BANDWIDTH_MAX;
        
        int bothLedsLit = 0;
        if ((rxLedsLit + txLedsLit) > PIXEL_COUNT) {
            bothLedsLit = (rxLedsLit + txLedsLit) - PIXEL_COUNT;
            rxLedsLit -= bothLedsLit;
            txLedsLit -= bothLedsLit;
        }
        
        for (int i = 0; i < PIXEL_COUNT; i++) {
            if (i < rxLedsLit) {
                strip.setPixelColor(i, RX_COLOUR);
            } else if (i < (rxLedsLit + bothLedsLit)) {
                strip.setPixelColor(i, RXTX_COLOUR);
            } else if (i >= (PIXEL_COUNT - txLedsLit)) {
                strip.setPixelColor(i, TX_COLOUR);
            } else {
                strip.setPixelColor(i, NONE_COLOUR);
            }
        }
        
        strip.show();
        
        Udp.flush();
    } else if (packetSize > 0) {
        Udp.flush();
    }
}

