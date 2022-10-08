`ntpescape` threat model and security policy
============================================

Threat model
------------

1. An attacker may be able to exfiltrate data from a compromised
   computer in a secure enclave to their own server through NTP. 

2. An attacker may be able to deny NTP services to other legitimate
   users in the network if network security responds too harshly and 
   blocks NTP.

3. (In future implementations) an attacker may be able to _infiltrate_
   data from a server to a compromised computer using NTP.

4. An attacker may be able to compromise a real, legitimate NTP server
   and use it as a receiver for NTP exfiltration. 

Security policy
---------------

**1 - NTP exfiltration risk**
   * a) Host internal NTP servers and block all outbound NTP traffic.
        Allow updates from these trusted internal servers to external
        NTP servers only.
   * b) Have a strict allow-list of NTP servers that any machine,
        including internal NTP servers, can talk to. 
   * c) Monitor for abnormal NTP behavior at the firewall / IDS. 
        `ntpescape` attempts to make this difficult, but you could 
        alarm on the following behaviors:
      * Abnormally high NTP client packet frequency.
      * Abnormally random NTP client packet frequency - `ntpescape` 
         attempts to simulate normal NTP daemon polling frequency, 
         but the true NTP daemon is more complicated and an advanced
         IDS may be able to tell the difference.
      * Incorrect NTP packets or extra data filled in NTP packets.
        `ntpescape` packets do not have extra data and are always
        legitimate, but other tools may use malformed packets to 
        increase data transfer rate.
      * Hardcoded response data - `ntpescape` responses are randomized
        and contain plausible, simulated response data. However, 
        other tools may return responses with hardcoded data in 
        naturally-varying fields such as the root dispersion and more.

**2 - NTP denial of service through harsh security response**
   * a) Ensure there is a way to make NTP queries at all times for
        computers not under investigation. This could be through
        NTP queries to internal NTP servers or certain allow-listed
        servers as in policy 1b.

**3 - NTP infiltration risk**
   * a) Follow all policies from 1a-1c. This is because data
        could be infiltrated using the lower random bits of the response
        timestamps in a way that would be as hard to detect as the 
        original outbound transmissions.
   * b) In addition to the IDS rules in 1c, alarm on unsolicited
        NTP responses from servers. This would be an easy indicator
        of data infiltration.

**4 - Legitimate NTP server compromise**
   * a) Update NTP server allow-lists frequently based on recent news.
        If an NTP server is compromised, remove it from the allow-list
        immediately.