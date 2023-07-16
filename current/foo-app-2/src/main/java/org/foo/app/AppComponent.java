/*
 * Copyright 2015 Open Networking Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */
package org.foo.app;

import com.google.common.collect.HashMultimap;
import org.onlab.packet.Ethernet;
import org.onlab.packet.IPv4;
import org.onlab.packet.MacAddress;
import org.onosproject.core.ApplicationId;
import org.onosproject.core.CoreService;
import org.onosproject.net.DeviceId;
import org.onosproject.net.PortNumber;
import org.onosproject.net.flow.DefaultFlowRule;
import org.onosproject.net.flow.DefaultTrafficSelector;
import org.onosproject.net.flow.DefaultTrafficTreatment;
import org.onosproject.net.flow.FlowRule;
import org.onosproject.net.flow.FlowRuleService;
import org.onosproject.net.flow.criteria.PiCriterion;
import org.onosproject.net.packet.PacketContext;
import org.onosproject.net.packet.PacketPriority;
import org.onosproject.net.packet.PacketProcessor;
import org.onosproject.net.packet.PacketService;
import org.onosproject.net.pi.model.PiActionId;
import org.onosproject.net.pi.model.PiMatchFieldId;
import org.onosproject.net.pi.model.PiTableId;
import org.onosproject.net.pi.runtime.PiAction;
import org.osgi.service.component.annotations.Activate;
import org.osgi.service.component.annotations.Component;
import org.osgi.service.component.annotations.Deactivate;
import org.osgi.service.component.annotations.Reference;
import org.osgi.service.component.annotations.ReferenceCardinality;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.util.ArrayList;
import java.util.List;
import java.util.Objects;
import java.util.Optional;

import javax.ws.rs.client.Client;
import javax.ws.rs.client.ClientBuilder;
import javax.ws.rs.client.Entity;
import javax.ws.rs.client.WebTarget;
import javax.ws.rs.core.HttpHeaders;
import javax.ws.rs.core.MediaType;
import javax.ws.rs.core.Response;

/**
 * Sample application that permits only one ICMP ping per minute for a unique
 * src/dst MAC pair per switch.
 */
@Component(immediate = true)
public class AppComponent {

    private static List<FirewallRule> firewallRules = new ArrayList<FirewallRule>();
    private static Logger log = LoggerFactory.getLogger(AppComponent.class);

    private static final int PRIORITY = 128;
    private static final int DROP_PRIORITY = 129;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected CoreService coreService;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected FlowRuleService flowRuleService;

    @Reference(cardinality = ReferenceCardinality.MANDATORY)
    protected PacketService packetService;

    private ApplicationId appId;
    private final PacketProcessor packetProcessor = new PingPacketProcessor();

    // Selector for traffic that is to be intercepted
    PiCriterion intercept = PiCriterion.builder()
            .matchTernary(PiMatchFieldId.of("hdr.ethernet.ether_type"), Ethernet.TYPE_IPV4, 0xffff)
            .matchTernary(PiMatchFieldId.of("hdr.ipv4.protocol"), IPv4.PROTOCOL_ICMP, 0xff)
            .build();

    // Means to track detected pings from each device on a temporary basis
    private final HashMultimap<DeviceId, PingRecord> pings = HashMultimap.create();

    @Activate
    public void activate() {
        appId = coreService.registerApplication("org.foo.app",
                () -> log.info("Periscope down."));
        packetService.addProcessor(packetProcessor, PRIORITY);
        packetService.requestPackets(DefaultTrafficSelector.builder().matchPi(intercept).build(),
                PacketPriority.CONTROL, appId, Optional.empty());
        log.info("Started");
    }

    @Deactivate
    public void deactivate() {
        packetService.removeProcessor(packetProcessor);
        flowRuleService.removeFlowRulesById(appId);
        log.info("Stopped");
    }

    // Processes the specified ICMP ping packet.
    private void processPing(PacketContext context, Ethernet eth, Byte proto) {
        DeviceId deviceId = context.inPacket().receivedFrom().deviceId();
        String src = eth.getSourceMAC().toStringNoColon();
        String dst = eth.getDestinationMAC().toStringNoColon();
        PingRecord ping = new PingRecord(eth.getSourceMAC(), eth.getDestinationMAC());

        for (FirewallRule firewallRule : firewallRules) {
            if (firewallRule.matches(src, dst) && Byte.valueOf(firewallRule.getProto()) == proto) {
                banPings(deviceId, eth.getSourceMAC(), eth.getDestinationMAC(), Byte.valueOf(firewallRule.getProto()));
                context.block();
                break;
            } else {
                pings.put(deviceId, ping);
            }
        }

    }

    // Installs a drop rule for the ICMP pings between given src/dst.
    private void banPings(DeviceId deviceId, MacAddress src, MacAddress dst, byte protocol) {
        PiCriterion match = PiCriterion.builder()
                .matchTernary(PiMatchFieldId.of("hdr.ethernet.ether_type"), Ethernet.TYPE_IPV4, 0xffff)
                .matchTernary(PiMatchFieldId.of("hdr.ipv4.protocol"), protocol, 0xff)
                .matchTernary(PiMatchFieldId.of("hdr.ethernet.src_addr"), src.toLong(), 0xffffffffffffL)
                .matchTernary(PiMatchFieldId.of("hdr.ethernet.dst_addr"), dst.toLong(), 0xffffffffffffL)
                .build();

        PiAction action = PiAction.builder()
                .withId(PiActionId.of("ingress.table0_control.drop"))
                .build();

        FlowRule dropRule = DefaultFlowRule.builder()
                .forDevice(deviceId).fromApp(appId).makePermanent().withPriority(
                        DROP_PRIORITY)
                .forTable(PiTableId.of("ingress.table0_control.table0"))
                .withSelector(DefaultTrafficSelector.builder().matchPi(match).build())
                .withTreatment(DefaultTrafficTreatment.builder().piTableAction(action).build())
                .build();

        // Apply the drop rule...
        flowRuleService.applyFlowRules(dropRule);

    }

    private static String getBasicAuthHeader() {
        String credentials = "onos" + ":" + "rocks";
        String base64Credentials = java.util.Base64.getEncoder().encodeToString(credentials.getBytes());
        return "Basic " + base64Credentials;
    }

    public void setRule(String source, String dest, Byte protocol) {
        FirewallRule f = new FirewallRule(source, dest, protocol, null, null);
        for (FirewallRule firewallRule : firewallRules) {
            // rule exists
            if (firewallRule.matches(source, dest) && Byte.valueOf(firewallRule.getProto()) == Byte.valueOf(protocol)) {
                return;
            } else {
                continue;
            }
        }
        firewallRules.add(f);
    }

    public void setRule(DeviceId deviceId, PortNumber port) throws Exception {
        FirewallRule f = new FirewallRule(null, null, null, deviceId, port);
        for (FirewallRule firewallRule : firewallRules) {
            // Rule already exists
            if (firewallRule.getDeviceId().equals(deviceId.toString()) && firewallRule.getPortNumber().equals(port.toString())) {
                return;
            }
        }
        firewallRules.add(f);
        Client client = ClientBuilder.newClient();
        WebTarget target = client.target("http://localhost:8181/onos/v1");
        String did = deviceId.toString();
        String p = port.toString();
        String endpoint = String.format("/devices/%s/portstate/%s", did, p);
        String requestBody = "{\"enabled\":false}";
        Response response = target.path(endpoint)
                .request(MediaType.APPLICATION_JSON)
                .header(HttpHeaders.AUTHORIZATION, getBasicAuthHeader())
                .post(Entity.json(requestBody), Response.class);
        if (response.getStatusInfo().equals(Response.Status.OK)) {

        } else {
            String msg = String.format("Some error occurred %s %s", response.getStatusInfo().toString(),
                    Response.Status.OK.toString());
            throw new Exception(msg);
        }
    }

    public void removeRule(String source, String dest, Byte protocol) {
        for (FirewallRule firewallRule : firewallRules) {
            if (firewallRule.matches(source, dest) && Byte.valueOf(firewallRule.getProto()) == Byte.valueOf(protocol)) {
                firewallRules.remove(firewallRule);
                break;
            } else {
                continue;
            }
        }
    }

    public void removeRule(DeviceId deviceId, PortNumber port) throws Exception {
        for (FirewallRule firewallRule : firewallRules) {
            if (firewallRule.getDeviceId().equals(deviceId.toString()) && firewallRule.getPortNumber().equals(port.toString())) {
                firewallRules.remove(firewallRule);
                break;
            }
        }
        Client client = ClientBuilder.newClient();
        WebTarget target = client.target("http://localhost:8181/onos/v1");
        String did = deviceId.toString();
        String p = port.toString();
        String endpoint = String.format("/devices/%s/portstate/%s", did, p);
        String requestBody = "{\"enabled\":true}";
        Response response = target.path(endpoint)
                .request(MediaType.APPLICATION_JSON)
                .header(HttpHeaders.AUTHORIZATION, getBasicAuthHeader())
                .post(Entity.json(requestBody), Response.class);
        if (response.getStatusInfo().equals(Response.Status.OK)) {

        } else {
            String msg = String.format("Some error occurred %s %s", response.getStatusInfo().toString(),
                    Response.Status.OK.toString());
            throw new Exception(msg);
        }
    }

    public static List<FirewallRule> getRules() {
        return firewallRules;
    }

    // Indicates whether the specified packet corresponds to ICMP ping.
    private boolean isIcmpPing(Ethernet eth) {
        return eth.getEtherType() == Ethernet.TYPE_IPV4 &&
                ((IPv4) eth.getPayload()).getProtocol() == IPv4.PROTOCOL_ICMP;
    }

    private boolean isTcpPacket(Ethernet eth) {
        return eth.getEtherType() == Ethernet.TYPE_IPV4 &&
                ((IPv4) eth.getPayload()).getProtocol() == IPv4.PROTOCOL_TCP;
    }

    // Intercepts packets
    private class PingPacketProcessor implements PacketProcessor {
        @Override
        public void process(PacketContext context) {
            Ethernet eth = context.inPacket().parsed();
            if (isIcmpPing(eth)) {
                processPing(context, eth, IPv4.PROTOCOL_ICMP);
            } else if (isTcpPacket(eth)) {
                processPing(context, eth, IPv4.PROTOCOL_TCP);
            }
        }
    }

    // Record of a ping between two end-station MAC addresses
    private class PingRecord {
        private final MacAddress src;
        private final MacAddress dst;

        PingRecord(MacAddress src, MacAddress dst) {
            this.src = src;
            this.dst = dst;
        }

        @Override
        public int hashCode() {
            return Objects.hash(src, dst);
        }

        @Override
        public boolean equals(Object obj) {
            if (this == obj) {
                return true;
            }
            if (obj == null || getClass() != obj.getClass()) {
                return false;
            }
            final PingRecord other = (PingRecord) obj;
            return Objects.equals(this.src, other.src) && Objects.equals(this.dst, other.dst);
        }
    }

    public class FirewallRule {
        private String source;
        private String dest;
        private Byte proto;
        private DeviceId deviceId;
        private PortNumber port;

        public FirewallRule(String source, String dest, Byte protocol, DeviceId deviceId, PortNumber port) {
            this.source = source;
            this.dest = dest;
            this.proto = protocol;
            this.deviceId = deviceId;
            this.port = port;
        }

        public FirewallRule(FirewallRule f) {
            this.dest = f.dest;
            this.proto = f.proto;
            this.source = f.source;
            this.deviceId = f.deviceId;
            this.port = f.port;
        }

        public boolean matches(String src, String dest) {
            return this.source.equals(src) && this.dest.equals(dest);
        }

        public String getSource() {
            return this.source == null ? "" : this.source;
        }

        public String getDest() {
            return this.dest == null ? "" : this.dest;
        }

        public String getProto() {
            return this.proto == null ? "" : this.proto.toString();
        }

        public String getDeviceId() {
            return this.deviceId == null ? "" : this.deviceId.toString();
        }

        public String getPortNumber() {
            return this.port == null ? "" : this.port.toString();
        }

        @Override
        public String toString() {
            return "Src:" + this.getSource() + ", Dest:" + this.getDest() + ", Proto:" + this.getProto()
                    + ", Device ID:" + this.getDeviceId() + ", Port:" + this.getPortNumber();
        }
    }
}
