import spacy
import zmq


context = zmq.Context()
socket = context.socket(zmq.REP)
socket.bind("tcp://127.0.0.1:5555")

while True:
    message = socket.recv_json()

    print("Message received: ", message)

    nlp = spacy.load("en_core_web_trf")
    doc = nlp(message["body"])

    reply = {}

    for ent in doc.ents:
        if ent.label_ == "ORG":
            reply["company"] = ent.text
            print("Found company: " + reply["company"])
            break

    socket.send_json(reply)

socket.close()
context.term()
