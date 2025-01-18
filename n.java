import io.netty.bootstrap.Bootstrap;
import io.netty.bootstrap.ServerBootstrap;
import io.netty.channel.*;
import io.netty.channel.nio.NioEventLoopGroup;
import io.netty.channel.socket.SocketChannel;
import io.netty.channel.socket.nio.NioServerSocketChannel;
import io.netty.channel.socket.nio.NioSocketChannel;

public class TcpProxy {

    private final String remoteHost;
    private final int remotePort;
    private final int localPort;

    public TcpProxy(String remoteHost, int remotePort, int localPort) {
        this.remoteHost = remoteHost;
        this.remotePort = remotePort;
        this.localPort = localPort;
    }

    public void run() throws Exception {
        EventLoopGroup bossGroup = new NioEventLoopGroup();
        EventLoopGroup workerGroup = new NioEventLoopGroup();
        try {
            ServerBootstrap b = new ServerBootstrap();
            b.group(bossGroup, workerGroup)
             .channel(NioServerSocketChannel.class)
             .childHandler(new ChannelInitializer<SocketChannel>() {
                 @Override
                 public void initChannel(SocketChannel ch) throws Exception {
                     ch.pipeline().addLast(new ClientToServerHandler());
                 }
             });

            ChannelFuture f = b.bind(localPort).sync();
            System.out.println("TCP Proxy started at port " + localPort);
            f.channel().closeFuture().sync();
        } finally {
            workerGroup.shutdownGracefully();
            bossGroup.shutdownGracefully();
        }
    }

    private class ClientToServerHandler extends ChannelInboundHandlerAdapter {

        private Channel outboundChannel;

        @Override
        public void channelActive(ChannelHandlerContext ctx) throws Exception {
            Bootstrap b = new Bootstrap();
            b.group(ctx.channel().eventLoop())
             .channel(NioSocketChannel.class)
             .handler(new ChannelInitializer<SocketChannel>() {
                 @Override
                 public void initChannel(SocketChannel ch) throws Exception {
                     ch.pipeline().addLast(new ServerToClientHandler(ctx.channel()));
                 }
             });

            // Connect to the remote server.
            ChannelFuture f = b.connect(remoteHost, remotePort).sync();
            outboundChannel = f.channel();
        }

        @Override
        public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
            if (outboundChannel.isActive()) {
                outboundChannel.writeAndFlush(msg);
            }
        }

        @Override
        public void channelInactive(ChannelHandlerContext ctx) throws Exception {
            if (outboundChannel != null) {
                closeOnFlush(outboundChannel);
            }
        }

        @Override
        public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) {
            cause.printStackTrace();
            closeOnFlush(ctx.channel());
        }
    }

    private class ServerToClientHandler extends ChannelInboundHandlerAdapter {

        private final Channel inboundChannel;

        public ServerToClientHandler(Channel inboundChannel) {
            this.inboundChannel = inboundChannel;
        }

        @Override
        public void channelRead(ChannelHandlerContext ctx, Object msg) throws Exception {
            inboundChannel.writeAndFlush(msg);
        }

        @Override
        public void channelInactive(ChannelHandlerContext ctx) throws Exception {
            closeOnFlush(inboundChannel);
        }

        @Override
        public void exceptionCaught(ChannelHandlerContext ctx, Throwable cause) {
            cause.printStackTrace();
            closeOnFlush(ctx.channel());
        }
    }

    static void closeOnFlush(Channel ch) {
        if (ch.isActive()) {
            ch.writeAndFlush(Unpooled.EMPTY_BUFFER).addListener(ChannelFutureListener.CLOSE);
        }
    }

    public static void main(String[] args) throws Exception {
        // Replace with your target host and ports.
        new TcpProxy("target.host", 80, 8080).run();
    }
}