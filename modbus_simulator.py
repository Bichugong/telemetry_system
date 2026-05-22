#!/usr/bin/env python3
# modbus_simulator.py
import serial
import time

# Настройки порта Moxa
PORT = "/dev/ttyUSB0"
BAUD = 9600

try:
    ser = serial.Serial(PORT, BAUD, timeout=1)
    print(f"Modbus Simulator started on {PORT}")
    print(f"Waiting for Modbus requests...")
    
    while True:
        # Ждём запрос от ESP32 (минимум 8 байт для Modbus)
        if ser.in_waiting >= 8:
            request = ser.read(8)
            
            # Проверка: запрос на чтение регистров (функция 03)
            if request[0] == 0x01 and request[1] == 0x03:
                # Ответ: Slave ID, Функция, Байты, Данные (2 байта), CRC
                # Пример: температура 25.0°C = 0x00FA = 250
                response = bytes([0x01, 0x03, 0x02, 0x00, 0xFA, 0xB9, 0x88])
                ser.write(response)
                print(f"Sent Modbus response: 25.0C")
        
        time.sleep(0.1)

except KeyboardInterrupt:
    print("\nStopped by user")
except serial.SerialException as e:
    print(f"Serial error: {e}")
    print("Check if port is busy: sudo lsof /dev/ttyUSB0")
except Exception as e:
    print(f"Error: {e}")
finally:
    if 'ser' in locals():
        ser.close()
