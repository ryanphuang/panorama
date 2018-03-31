package edu.jhu.order.deephealth;

import java.io.File;
import java.io.FileInputStream;
import java.io.IOException;
import java.util.Properties;
import java.util.logging.Logger;
import java.util.Map.Entry;

public class DHConfig
{
  private static final Logger logger = Logger.getLogger(DHConfig.class.getName());

  public static final int DEFAULT_PORT = 6688;
  public static final int DEFAULT_EXPIRE_MS = 3000;

  protected String dhServerAddr;
  protected int dhServerPort = DEFAULT_PORT;
  protected int expireMS = DEFAULT_EXPIRE_MS; // pending status expiration interval

  public String getDHServerAddr() { return dhServerAddr; }
  public int getDHServerPort() { return dhServerPort; }
  public int getExpireMs() { return expireMS; }

  public boolean parseProperties(Properties prop)
  {
    boolean success = false;
    for (Entry<Object, Object> entry : prop.entrySet()) {
      String key = entry.getKey().toString().trim();
      String value = entry.getValue().toString().trim();
      if(key.equals("dhserver")) {
        String parts[] = value.split(":");
        if (parts.length != 2) {
          logger.severe(value + " does not have the form host:port"); 
        }
        dhServerAddr = parts[0];
        dhServerPort = Integer.parseInt(parts[1]);
        success = true;
      } else if (key.equals("expire_ms")) {
        expireMS = Integer.parseInt(value);
        success = true;
      }
    }
    return success;
  }

  public static DHConfig parse(String path)
  {
    File configFile = new File(path);
    logger.info("Reading configuration from: " + configFile);
    try {
      if (!configFile.exists()) {
        throw new IllegalArgumentException(configFile.toString() + " file is missing");
      }
      Properties cfg = new Properties();
      FileInputStream in = new FileInputStream(configFile);
      try {
        cfg.load(in);
      } finally {
        in.close();
      }
      DHConfig config = new DHConfig();
      if (config.parseProperties(cfg)) {
        return config;
      } else {
        return null;
      }
    } catch (IOException e) {
      logger.severe("Error processing " + path + e);
      return null;
    } catch (IllegalArgumentException e) {
      logger.severe("Error processing " + path + e);
      return null;
    }
  }
}
