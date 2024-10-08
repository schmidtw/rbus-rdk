-- SPDX-FileCopyrightText: 2023 Comcast Cable Communications Management, LLC
-- SPDX-License-Identifier: Apache-2.0
--
-- In linux, place this file in ~/.local/lib/wireshark/plugins/
-- Details: https://www.wireshark.org/docs/wsug_html_chunked/ChPluginFolders.html
--
rbus_proto = Proto("rbus","rbus Postdissector")

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
post_F      = ProtoField.uint16("rbus.post",            "Post-amble",           base.HEX)
payload_F   = ProtoField.none(  "rbus.payload",         "Payload",              base.HEX)

-- add the field to the protocol
rbus_proto.fields = {pre_F, ver_F, hlen_F, seq_F, flags_F, ctrl_F, plen_F,
                     topiclen_F, topic_F, rtopiclen_F, rtopic_F, post_F, payload_F}

-- create a function to "postdissect" each frame
function rbus_proto.dissector(buffer,pinfo,tree)
    length = buffer:len()
    if length == 0 then return end
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
    offset = offset + tlen

    headerSt:add(rtopiclen_F, buffer(offset,4))
    local rtlen = buffer(offset,4):int()
    offset = offset + 4

    headerSt:add(rtopic_F, buffer(offset,rtlen))
    offset = offset + rtlen

    headerSt:add(post_F, buffer(offset,2))
    offset = offset + 2

    -- I don't think this is the best approach...
    payloadSt:add(payload_F, buffer(offset, length-offset))
end

tcp_table = DissectorTable.get("tcp.port")
-- Definitely not the best approach, but it works for now.
tcp_table:add(10001,rbus_proto)
