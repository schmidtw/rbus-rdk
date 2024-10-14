-- SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
-- SPDX-License-Identifier: Apache-2.0
--
-- In linux, place this file in ~/.local/lib/wireshark/plugins/
-- Details: https://www.wireshark.org/docs/wsug_html_chunked/ChPluginFolders.html
--

print("Loading rbus dissector")

rbus_proto = Proto("RBUS","rbus Dissector")

-- create the fields for our "protocol"
pre_F       = ProtoField.uint16("rbus.preamble",        "Pre-amble",            base.HEX)
ver_F       = ProtoField.uint16("rbus.version",         "Version",              base.HEX)
hlen_F      = ProtoField.uint16("rbus.header_length",   "Header Length",        base.Dec)
seq_F       = ProtoField.uint32("rbus.seq",             "Sequence Number",      base.HEX)
flags_F     = ProtoField.uint32("rbus.flags",           "Flags",                base.HEX)
ctrl_F      = ProtoField.uint32("rbus.ctrl",            "Control Data",         base.HEX)
plen_F      = ProtoField.uint32("rbus.payload_len",     "Payload Length",       base.DEC)
topiclen_F  = ProtoField.uint32("rbus.topic_len",       "Topic Length",         base.DEC)
topic_F     = ProtoField.string("rbus.topic",           "Topic"                         )
rtopiclen_F = ProtoField.uint32("rbus.reply_topic_len", "Reply Topic Length",   base.DEC)
rtopic_F    = ProtoField.string("rbus.reply_topic",     "Reply Topic"                   )
ts1_F       = ProtoField.uint32("rbus.ts1",             "Timestamp 1",          base.HEX)
ts2_F       = ProtoField.uint32("rbus.ts2",             "Timestamp 2",          base.HEX)
ts3_F       = ProtoField.uint32("rbus.ts3",             "Timestamp 3",          base.HEX)
ts4_F       = ProtoField.uint32("rbus.ts4",             "Timestamp 4",          base.HEX)
ts5_F       = ProtoField.uint32("rbus.ts5",             "Timestamp 5",          base.HEX)
post_F      = ProtoField.uint16("rbus.post",            "Post-amble",           base.HEX)
payload_F   = ProtoField.none(  "rbus.payload",         "Payload",              base.HEX)

-- add the field to the protocol
rbus_proto.fields = {pre_F, ver_F, hlen_F, seq_F, flags_F, ctrl_F, plen_F,
                     topiclen_F, topic_F, rtopiclen_F, rtopic_F,
                     ts1_F, ts2_F, ts3_F, ts4_F, ts5_F,
                     post_F, payload_F}

-- create a function to "postdissect" each frame
function rbus_proto.dissector(buffer,pinfo,tree)
    print("Dissecting packet")
    length = buffer:len()
    if length < 6 then
        print("Packet too short")
        return
    end

    -- Heuristic check: Ensure the packet has the expected preamble
    if buffer(0, 2):uint() ~= 0xaaaa then
        print("Invalid preamble")
        return
    end

    -- Check if we have enough data for the header
    local header_len = buffer(4, 2):uint() -- assuming header length is at offset 4
    if length < header_len then
        pinfo.desegment_len = header_len
        pinfo.desegment_offset = 0
        print("Not enough data for header")
        return
    end

    -- Check if we have enough data for the entire packet
    --local payload_len = buffer(18, 4):uint() -- assuming payload length is at offset 18
    --local total_len = header_len + payload_len
    --if length < total_len then
        --pinfo.desegment_len = total_len
        --pinfo.desegment_offset = 0
        --print("Not enough data for entire packet")
        --return
    --end

    pinfo.cols.protocol = rbus_proto.name

    local offset = 0
    local subtree = tree:add(rbus_proto, buffer(), "Rbus Packet")
    local headerSt = subtree:add(rbus_proto, buffer(), "Header")
    local payloadSt = subtree:add(rbus_proto, buffer(), "Payload")

    headerSt:add(pre_F, buffer(offset,2))
    offset = offset + 2

    headerSt:add(ver_F, buffer(offset,2))
    offset = offset + 2

    headerSt:add(hlen_F, buffer(offset,2))
    offset = offset + 2

    headerSt:add(seq_F, buffer(offset,4))
    offset = offset + 4

    headerSt:add(flags_F, buffer(offset,4))
    offset = offset + 4

    headerSt:add(ctrl_F, buffer(offset,4))
    offset = offset + 4

    headerSt:add(plen_F, buffer(offset,4))
    offset = offset + 4

    headerSt:add(topiclen_F, buffer(offset,4))
    local tlen = buffer(offset,4):int()
    offset = offset + 4

    headerSt:add(topic_F, buffer(offset,tlen))
end

-- Register the dissector
tcp_table=DissectorTable.get("tcp.port")
if tcp_table then
    tcp_table:add(10001, rbus_proto)
    print("rbus dissector registered with TCP port 10001")
else
    print("Failed to get tcp.port dissector table")
end