#include <Arduino.h>
#include <WiFi.h>
#include <PubSubClient.h>
#include <ArduinoJson.h>

#define WIFI_SSID "your_wifi_ssid"
#define WIFI_PASSWORD "your_wifi_password"
#define MQTT_SERVER "192.168.1.100"
#define MQTT_PORT 1883
#define MQTT_TOPIC "telemetry/esp32-gw-001/sensor"

#define RS485_DE_PIN 5
#define RS485_RE_PIN 4
#define RS485_RX_PIN 16
#define RS485_TX_PIN 17

#define FILTER_WINDOW 10
#define POLL_INTERVAL 5000

WiFiClient espClient;
PubSubClient mqttClient(espClient);

struct MovingAverage {
    float buffer[FILTER_WINDOW];
    uint8_t index;
    float sum;
    bool isFull;
    MovingAverage() : index(0), sum(0), isFull(false) { memset(buffer, 0, sizeof(buffer)); }
    float push(float value) {
        if (isFull) sum -= buffer[index];
        buffer[index] = value;
        sum += value;
        index = (index + 1) % FILTER_WINDOW;
        if (index == 0) isFull = true;
        uint8_t count = isFull ? FILTER_WINDOW : index;
        return count > 0 ? sum / count : value;
    }
};

MovingAverage tempFilter;
float previousValue = 0.0;
unsigned long lastPollTime = 0;

uint16_t crc16(const uint8_t* data, uint8_t len) {
    uint16_t crc = 0xFFFF;
    for (uint8_t i = 0; i < len; i++) {
        crc ^= data[i];
        for (uint8_t j = 0; j < 8; j++) {
            if (crc & 0x0001) { crc >>= 1; crc ^= 0xA001; }
            else { crc >>= 1; }
        }
    }
    return crc;
}

bool readModbusReg(uint8_t slaveId, uint16_t regAddress, uint16_t* value, uint32_t timeout = 1000) {
    uint8_t req[8];
    req[0] = slaveId;
    req[1] = 0x03;
    req[2] = highByte(regAddress);
    req[3] = lowByte(regAddress);
    req[4] = 0x00;
    req[5] = 0x01;
    uint16_t c = crc16(req, 6);
    req[6] = lowByte(c);
    req[7] = highByte(c);

    digitalWrite(RS485_DE_PIN, HIGH);
    digitalWrite(RS485_RE_PIN, HIGH);
    delayMicroseconds(100);
    Serial2.write(req, 8);
    Serial2.flush();
    digitalWrite(RS485_DE_PIN, LOW);
    digitalWrite(RS485_RE_PIN, LOW);

    uint8_t res[7];
    uint8_t idx = 0;
    unsigned long start = millis();
    while (millis() - start < timeout) {
        if (Serial2.available()) {
            res[idx++] = Serial2.read();
            if (idx >= 7) break;
        }
    }
    if (idx < 7) return false;

    uint16_t calcCrc = crc16(res, 5);
    uint16_t recvCrc = (uint16_t)res[6] << 8 | res[5];
    if (calcCrc != recvCrc) return false;

    *value = (uint16_t)res[3] << 8 | res[4];
    return true;
}

bool checkRange(float v, float min, float max) { return v >= min && v <= max; }
bool checkRate(float cur, float maxD) {
    float d = abs(cur - previousValue);
    previousValue = cur;
    return d <= maxD;
}

void connectWiFi() {
    Serial.print("Connecting WiFi");
    WiFi.begin(WIFI_SSID, WIFI_PASSWORD);
    int att = 0;
    while (WiFi.status() != WL_CONNECTED && att < 20) { delay(500); Serial.print("."); att++; }
    if (WiFi.status() == WL_CONNECTED) { Serial.println("\nWiFi OK"); Serial.println(WiFi.localIP()); }
    else Serial.println("\nWiFi FAIL");
}

void connectMQTT() {
    mqttClient.setServer(MQTT_SERVER, MQTT_PORT);
    while (!mqttClient.connected()) {
        Serial.print("Connecting MQTT...");
        if (mqttClient.connect("ESP32Gateway")) Serial.println("OK");
        else { Serial.print("FAIL "); Serial.println(mqttClient.state()); delay(5000); }
    }
}

void sendTelemetry(float temp, float pres, float cur) {
    if (!mqttClient.connected()) connectMQTT();
    StaticJsonDocument<256> doc;
    doc["sensor_id"] = "temp-01";
    doc["gateway_id"] = "esp32-gw-001";
    doc["temperature"] = temp;
    doc["pressure"] = pres;
    doc["current"] = cur;
    doc["timestamp"] = millis();
    doc["rssi"] = WiFi.RSSI();
    char buf[512];
    serializeJson(doc, buf);
    if (mqttClient.publish(MQTT_TOPIC, buf)) Serial.println("Sent OK");
    else Serial.println("Sent FAIL");
}

void setup() {
    Serial.begin(115200);
    delay(1000);
    Serial.println("ESP32 Gateway Start");
    pinMode(RS485_DE_PIN, OUTPUT);
    pinMode(RS485_RE_PIN, OUTPUT);
    digitalWrite(RS485_DE_PIN, LOW);
    digitalWrite(RS485_RE_PIN, LOW);
    Serial2.begin(9600, SERIAL_8N1, RS485_RX_PIN, RS485_TX_PIN);
    connectWiFi();
    connectMQTT();
    Serial.println("Ready");
}

void loop() {
    mqttClient.loop();
    if (WiFi.status() != WL_CONNECTED) connectWiFi();
    if (millis() - lastPollTime >= POLL_INTERVAL) {
        lastPollTime = millis();
        uint16_t raw = 0;
        if (readModbusReg(1, 0, &raw)) {
            float val = raw / 10.0;
            float filt = tempFilter.push(val);
            if (!checkRange(filt, 0.0, 100.0)) { Serial.println("Range ERR"); return; }
            if (!checkRate(filt, 15.0)) { Serial.println("Rate ERR"); return; }
            sendTelemetry(filt, 0.0, 0.0);
        } else {
            Serial.println("Modbus ERR");
        }
    }
    delay(100);
}