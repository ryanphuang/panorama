package edu.jhu.order.deephealth;

import io.grpc.ManagedChannel;
import io.grpc.ManagedChannelBuilder;
import io.grpc.StatusRuntimeException;
import io.grpc.stub.StreamObserver;
import com.google.protobuf.Message;

import java.io.IOException;
import java.io.InputStream;
import java.net.InetAddress;
import java.net.InetSocketAddress;
import java.net.UnknownHostException;
import java.text.DateFormat;
import java.text.SimpleDateFormat;
import java.util.Date;
import java.util.HashMap;
import java.util.Map;
import java.util.concurrent.TimeUnit;
import java.util.logging.ConsoleHandler;
import java.util.logging.LogManager;
import java.util.logging.LogRecord;
import java.util.logging.Logger;
import java.util.logging.SimpleFormatter;

import edu.jhu.order.deephealth.Health.Observation;
import edu.jhu.order.deephealth.Health.Report;
import edu.jhu.order.deephealth.DHBuffer.AggregateValue;
import edu.jhu.order.deephealth.Health.Metric;
import edu.jhu.order.deephealth.HealthServiceGrpc.HealthServiceBlockingStub;
import edu.jhu.order.deephealth.HealthServiceGrpc.HealthServiceStub;
import edu.jhu.order.deephealth.Service.GetReportRequest;
import edu.jhu.order.deephealth.Service.ObserveReply;
import edu.jhu.order.deephealth.Service.ObserveRequest;
import edu.jhu.order.deephealth.Service.SubmitReportReply;
import edu.jhu.order.deephealth.Service.SubmitReportRequest;
import edu.jhu.order.deephealth.Service.PingRequest;
import edu.jhu.order.deephealth.Service.PingReply;
import edu.jhu.order.deephealth.Service.RegisterRequest;
import edu.jhu.order.deephealth.Service.RegisterReply;
import edu.jhu.order.deephealth.Service.Peer;

import com.google.protobuf.util.Timestamps;

public class DHClient
{
	private static final Logger logger;
  static {
    Logger mainLogger = Logger.getLogger("edu.jhu.order.deephealth");
    mainLogger.setUseParentHandlers(false);
    ConsoleHandler handler = new ConsoleHandler();
    handler.setFormatter(new SimpleFormatter() {
      private static final String format = "%1$tF %1$tH:%1$tM:%1$tS,%1$tL - %2$-5s [%3$s] %4$s%n";
      @Override
      public synchronized String format(LogRecord lr) {
        return String.format(format,
            new Date(lr.getMillis()),
            lr.getLevel().getLocalizedName(),
            lr.getLoggerName(),
            lr.getMessage()
            );
      }
    });
    mainLogger.addHandler(handler);
    logger = Logger.getLogger(DHClient.class.getName());
  }

  public static final int DEFAULT_PORT = 6688;

  private String module;
  private String id;
  private String serverAddr;
  private int serverPort;
  private long handle;

  private ManagedChannel channel;
  private HealthServiceBlockingStub blockingStub;
  private HealthServiceStub asyncStub;
  private boolean ready = false;

  private final DHRateLimiter rateLimiter;

  private final Map<String, InetSocketAddress> subjectToAddress = new HashMap<String, InetSocketAddress>();
  private final Map<String, String> ipPortToSubject  = new HashMap<String, String>();
  private final Map<String, String> ipToSubject  = new HashMap<String, String>();
  private final Map<String, String> hostToSubject  = new HashMap<String, String>();

  private static DHClient instance = null;

  public static DHClient getInstance() {
    if (instance == null) {
      instance = new DHClient();
    }
    return instance;
  }

  private DHClient()
  {
    rateLimiter = new DHRateLimiter();
  }

  public void mapSubject(String subject, InetSocketAddress address)
  {
    subjectToAddress.put(subject, address);
    ipPortToSubject.put(address.toString(), subject);
    ipToSubject.put(address.getAddress().getHostAddress(), subject);
    hostToSubject.put(address.getHostName(), subject);
    logger.info("Map " + subject + " to " + address.getHostName() + "/" + address.getAddress().getHostAddress());
  }

  public boolean init(String module, String id)
  {
    try {
      String hostname = InetAddress.getLocalHost().getHostName().split("\\.")[0];
      return init(hostname, DEFAULT_PORT, module, id);
    } catch (UnknownHostException e) {
      logger.warning("Failed to infer host name: " + e);
      return false;
    }
  }

  public boolean init(String addr, int port, String module, String id)
  {
    if (ready)
      return true;

    this.serverAddr = addr;
    this.serverPort = port;
    this.module = module;
    this.id = id;

    ManagedChannelBuilder<?> channelBuilder = ManagedChannelBuilder.forAddress(
        serverAddr, serverPort).usePlaintext(true);
    this.channel = channelBuilder.build();
    this.blockingStub = HealthServiceGrpc.newBlockingStub(channel);
    this.asyncStub = HealthServiceGrpc.newStub(channel);

    RegisterRequest request = RegisterRequest.newBuilder().setModule(this.module).setObserver(id).build();
    RegisterReply reply;
    try {
      reply = blockingStub.register(request);
    } catch (StatusRuntimeException e) {
      logger.warning("Register RPC failed: " + e.getStatus());
      return false;
    }
    handle = reply.getHandle();
    logger.info("Got register reply with handle " + handle);
    this.ready = true;
    return true;
  }

  public String getServerAddr()
  {
    return serverAddr;
  }

  public int getServerPort()
  {
    return serverPort;
  }

  public String getId()
  {
    return id;
  }

  public void setId(String ID)
  {
    id = ID;
  }

  public void shutdown() throws InterruptedException 
  {
    if (!ready)
      return;
    channel.shutdown().awaitTermination(5, TimeUnit.SECONDS);
    ready = false;
  }

  public HealthServiceBlockingStub block()
  {
    return blockingStub;
  }

  public HealthServiceStub async()
  {
    return asyncStub;
  }

  public boolean observe(String subject)
  {
    if (!ready)
      return false;
    logger.info("Start observing " + subject);
    ObserveRequest request = ObserveRequest.newBuilder().setSubject(subject).build();
    ObserveReply reply;
    try {
      reply = blockingStub.observe(request);
    } catch (StatusRuntimeException e) {
      logger.warning("Observe RPC failed: " + e.getStatus());
      return false;
    }
    boolean ok = reply.getSuccess();
    logger.info("Result: " + ok);
    return ok;
  }

  public boolean stopObserving(String subject)
  {
    if (!ready)
      return false;
    logger.info("Stop observing " + subject);
    ObserveRequest request = ObserveRequest.newBuilder().setSubject(subject).build();
    ObserveReply reply; 
    try {
      reply = blockingStub.stopObserving(request);
    } catch (StatusRuntimeException e) {
      logger.warning("StopObserving RPC failed: " + e.getStatus());
      return false;
    }
    boolean ok = reply.getSuccess();
    logger.info("Result: " + ok);
    return ok;
  }

  private static final StreamObserver<SubmitReportReply> gEmptySubmityResponseObserver = new StreamObserver<SubmitReportReply>() {
      @Override
      public void onNext(SubmitReportReply reply) {
      }
      @Override
      public void onError(Throwable t) {
      }
      @Override
      public void onCompleted() {
      }
  };

  public void selfInform(String name, Health.Status status, float score)
  {
    inform(this.id, name, status, score, false);
  }

  public void selfInformAsync(String name, Health.Status status, float score)
  {
    informAsync(this.id, name, status, score, false);
  }

  public void inform(String subject, String name, Health.Status status, float score, boolean resolve)
  {
    if (resolve) {
      String resolved = ipToSubject.get(subject);
      if (resolved != null)
        subject = resolved;
    }
    Metric metric = rateLimiter.vet(subject, name, status, score);
    if (metric != null) {
      report(subject, metric);
    }
  }

  public void informAsync(String subject, String name, Health.Status status, float score, boolean resolve)
  {
    if (resolve) {
      String resolved = ipToSubject.get(subject);
      if (resolved != null)
        subject = resolved;
    }
    Metric metric = rateLimiter.vet(subject, name, status, score);
    if (metric != null) {
      reportAsync(null, subject, metric);
    }
  }

  public SubmitReportReply.Status selfReport(Metric... metrics)
  {
    return report(this.id, metrics);
  }

  public SubmitReportReply.Status report(String subject, Metric... metrics)
  {
    if (!ready)
      return null;
    long timeMillis = System.currentTimeMillis();
    logger.info("Submitting report from " + id + " about " + subject + " at " + timeMillis);
    Observation observation = DHBuilder.NewObservation(timeMillis, metrics);
    Report report = Report.newBuilder().setObserver(id).setSubject(subject).setObservation(observation).build();
    SubmitReportRequest request = SubmitReportRequest.newBuilder().setHandle(handle).setReport(report).build();
    SubmitReportReply reply; 
    try {
      reply = blockingStub.submitReport(request);
    } catch (StatusRuntimeException e) {
      logger.warning("SubmitReport RPC failed: " + e.getStatus());
      return null;
    }
    SubmitReportReply.Status status = reply.getResult();
    logger.info("Result: " + status);
    return status;
  }

  public void reportAsync(final AsyncCallBack cb, String subject, Metric... metrics) 
  {
    if (!ready)
      return;
    long timeMillis = System.currentTimeMillis();
    logger.info("Asynchronously submitting report from " + id + " about " + subject + " at " + timeMillis);
    Observation observation = DHBuilder.NewObservation(timeMillis, metrics);
    Report report = Report.newBuilder().setObserver(id).setSubject(subject).setObservation(observation).build();
    SubmitReportRequest request = SubmitReportRequest.newBuilder().setHandle(handle).setReport(report).build();
    SubmitReportReply reply; 

    StreamObserver<SubmitReportReply> responseObserver;
    if (cb == null) {
      responseObserver = gEmptySubmityResponseObserver;
    } else {
      responseObserver = new StreamObserver<SubmitReportReply>() {
        @Override
        public void onNext(SubmitReportReply reply) {
          logger.info("Got async submit report reply: " + reply);
          cb.onMessage(reply);
        }
        @Override
        public void onError(Throwable t) {
          logger.info("Async submit report error: " + t);
          cb.onRpcError(t);
        }
        @Override
        public void onCompleted() {
          cb.onCompleted();
        }
      };
    }
    asyncStub.submitReport(request, responseObserver);
  }

  public Report getReport(String subject)
  {
    if (!ready)
      return null;
    logger.info("Getting report for " + subject);
    GetReportRequest request = GetReportRequest.newBuilder().setSubject(subject).build();
    Report report;
    try {
      report = blockingStub.getLatestReport(request);
    } catch (StatusRuntimeException e) {
      logger.warning("GetLatestReport RPC failed: " + e.getStatus());
      return null;
    }
    logger.info("Result: " + report);
    return report;
  }

  // ping local health server
  public long ping()
  {
    if (!ready)
      return -1;
    long timeMillis = System.currentTimeMillis();
    Peer source = Peer.newBuilder().setId(id).setAddr("localhost").build();
    PingRequest request = PingRequest.newBuilder().setSource(source).setTime(Timestamps.fromMillis(timeMillis)).build();
    PingReply reply;
    try {
      reply = blockingStub.ping(request);
    } catch (StatusRuntimeException e) {
      logger.warning("Ping RPC failed: " + e.getStatus());
      return -1;
    }
    long result = Timestamps.toMillis(reply.getTime());
    Date date = new Date(result);
    DateFormat formatter = new SimpleDateFormat("HH:mm:ss:SSS");
    logger.info("Got ping reply with time: " + formatter.format(date));
    return result;
  }

  public interface AsyncCallBack {
    void onMessage(Message message);
    void onRpcError(Throwable exception);
    void onCompleted();
  }

  public class SubjectAddrEntry {
    public String subject;
    public InetSocketAddress address;
    public SubjectAddrEntry(String subject, InetSocketAddress address) {
      this.subject = subject;
      this.address = address;
    }
  }
}
