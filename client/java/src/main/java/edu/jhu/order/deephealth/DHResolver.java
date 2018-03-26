package edu.jhu.order.deephealth;

import java.util.logging.Logger;
import java.util.HashMap;
import java.util.Map;

import java.net.InetAddress;
import java.net.InetSocketAddress;

public class DHResolver {

	private static final Logger logger = Logger.getLogger(DHClient.class.getName());

  protected final Map<String, InetSocketAddress> subjectToAddress = new HashMap<String, InetSocketAddress>();
  protected final Map<String, String> ipPortToSubject  = new HashMap<String, String>();
  protected final Map<String, String> ipToSubject  = new HashMap<String, String>();
  protected final Map<String, String> hostToSubject  = new HashMap<String, String>();

  public enum RType {
    IP,
    IP_PORT,
    HOST
  }

  public void map(String subject, InetSocketAddress address)
  {
    subjectToAddress.put(subject, address);
    ipPortToSubject.put(address.toString(), subject);
    ipToSubject.put(address.getAddress().getHostAddress(), subject);
    hostToSubject.put(address.getHostName(), subject);
    logger.info("Map " + subject + " to " + address.getHostName() + "/" + address.getAddress().getHostAddress());
  }

  public String resolve(String subject, RType rt)
  {
    switch (rt) {
      case IP:
        return ipToSubject.get(subject);
      case IP_PORT:
        return ipPortToSubject.get(subject);
      case HOST:
        return hostToSubject.get(subject);
      default:
        return null;
    }
  }

  public static class SubjectAddrEntry {
    public String subject;
    public InetSocketAddress address;
    public SubjectAddrEntry(String subject, InetSocketAddress address) {
      this.subject = subject;
      this.address = address;
    }
  }
}
