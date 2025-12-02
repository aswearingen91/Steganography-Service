# Steganography-Service

## Known Limitations

* Only accepts PNG and JPEG files.
* The size of a message is limited by the size of the image that contains it.

## Running This Program

This service accepts a single environment variable "PORT". If not set it defaults to 8080.

````azure
# set custom port
PORT=9090

go run cmd/stegsvc/main.go
````

## Example Requests 

### Encode

````
import base64
import json
import requests

BASE_URL = "http://localhost:8080"   # Change if deployed


def encode_image(image_path: str, message: str):
    """
    Sends an image + message to /encode and receives a Base64 PNG.
    """
    url = f"{BASE_URL}/encode"
    print(f"[encode] POST {url}")

    with open(image_path, "rb") as f:
        files = {
            "image": (image_path, f, "application/octet-stream")
        }
        data = {
            "message": message
        }

        resp = requests.post(url, files=files, data=data)

    print(f"[encode] Status: {resp.status_code}")

    if resp.status_code != 200:
        print("[encode] Error:", resp.text)
        return None

    payload = resp.json()
    print("[encode] Server Response JSON:\n", json.dumps(payload, indent=2))

    # Convert returned base64 PNG into file
    out_png_bytes = base64.b64decode(payload["imageBase64"])
    output_filename = payload.get("filename", "encoded_output.png")

    with open(output_filename, "wb") as out:
        out.write(out_png_bytes)

    print(f"[encode] Saved encoded image as: {output_filename}")
    return output_filename


if __name__ == "__main__":
    # Customize these
    input_image = "example.png"
    secret_message = "Hello, steganography!"

    encode_image(input_image, secret_message)

````

### Decode

````python
import json
import requests

BASE_URL = "http://localhost:8080"   # Change if deployed


def decode_image(image_path: str):
    """
    Sends an image to /decode and receives the hidden message.
    """
    url = f"{BASE_URL}/decode"
    print(f"[decode] POST {url}")

    with open(image_path, "rb") as f:
        files = {
            "image": (image_path, f, "application/octet-stream")
        }

        resp = requests.post(url, files=files)

    print(f"[decode] Status: {resp.status_code}")

    if resp.status_code != 200:
        print("[decode] Error:", resp.text)
        return None

    payload = resp.json()
    print("[decode] Hidden message:", payload["message"])
    return payload["message"]


if __name__ == "__main__":
    # Customize this
    encoded_image = "steg_image.png"  # file returned from encode

    decode_image(encoded_image)

````


